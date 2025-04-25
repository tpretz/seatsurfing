package messaging

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/config"
	. "github.com/seatsurfing/seatsurfing/server/repository"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// SlackApp manages the Slack bot connection and event handling
type SlackApp struct {
	client       *slack.Client
	socketClient *socketmode.Client
	cancel       context.CancelFunc
	config       *Config
	mu           sync.Mutex
	wg           sync.WaitGroup
}

// SlackError represents a contextual error
type SlackError struct {
	Err     error
	Context string
}

func (e *SlackError) Error() string {
	return fmt.Sprintf("%s: %v", e.Context, e.Err)
}

var (
	slackAppInstance *SlackApp
	slackOnce        sync.Once
)

// NewSlackApp creates a new SlackApp instance with the given configuration
func InitializeSlackClient() *SlackApp {
	cfg := GetConfig()
	slackOnce.Do(func() {
		slackAppInstance = &SlackApp{
			config: cfg,
		}
		slackAppInstance.start()
	})
	return slackAppInstance
}

// start initializes the Slack connection with retry logic
func (s *SlackApp) start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i <= s.config.SlackRetryCount; i++ {
		if err := s.connect(); err == nil {
			return
		}
		log.Printf("Connection attempt %d failed, retrying in %v", i+1, s.config.SlackRetryBackoff)
		time.Sleep(s.config.SlackRetryBackoff)
	}
	log.Fatal("Failed to connect to Slack after retries")
}

func (s *SlackApp) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.client = slack.New(
		s.config.SlackAuthToken,
		slack.OptionDebug(s.config.SlackDebug),
		slack.OptionAppLevelToken(s.config.SlackAppToken),
	)

	s.socketClient = socketmode.New(
		s.client,
		socketmode.OptionDebug(s.config.SlackDebug),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	// Store the cancellable context for shutdown
	ctx, s.cancel = context.WithCancel(context.Background())

	// Run the socket client in a goroutine with context
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.socketClient.RunContext(ctx); err != nil {
			log.Printf("Socket client run error: %v", err)
		}
	}()

	// Start the event loop in a separate goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runEventLoop(ctx)
	}()

	return nil // Initial connection setup doesn’t block here
}

// runEventLoop handles Slack events
func (s *SlackApp) runEventLoop(ctx context.Context) {
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Println("Shutting down socketmode listener")
				return
			case event := <-s.socketClient.Events:
				s.wg.Add(1)
				go func(e socketmode.Event) {
					defer s.wg.Done()
					if err := s.handleEvent(e); err != nil {
						log.Printf("Error handling event: %v", err)
					}
				}(event)
			}
		}
	}()
}

// handleEvent processes individual Slack events
func (s *SlackApp) handleEvent(event socketmode.Event) error {
	switch event.Type {
	case socketmode.EventTypeEventsAPI:
		eventsAPI, ok := event.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return &SlackError{Err: fmt.Errorf("invalid event type"), Context: "type assertion"}
		}
		s.socketClient.Ack(*event.Request)
		return s.handleEventMessage(eventsAPI)
	}
	return nil
}

// handleEventMessage processes Slack event messages
func (s *SlackApp) handleEventMessage(event slackevents.EventsAPIEvent) error {
	if event.Type != slackevents.CallbackEvent {
		return nil
	}

	switch evnt := event.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		return s.handleAppMentionEventToBot(evnt)
	default:
		log.Printf("Ignoring unknown inner event type: %T", event.InnerEvent.Data)
		return nil
	}
}

// handleAppMentionEventToBot processes app mention events
func (s *SlackApp) handleAppMentionEventToBot(event *slackevents.AppMentionEvent) error {
	user, err := s.client.GetUserInfo(event.User)
	if err != nil {
		return &SlackError{Err: err, Context: "getting user info"}
	}

	// Get Organization ID
	orgID, err := getOrgID(s.config.OrgName)
	if err != nil {
		return &SlackError{Err: err, Context: "OrgID cannot be fetched"}
	}

	userAD, err := GetUserRepository().GetByEmail(orgID, user.Profile.Email)
	if err != nil {
		return &SlackError{Err: err, Context: "getting user from repository"}
	}

	attachment, err := s.processCommand(strings.ToLower(event.Text), userAD)
	if err != nil {
		return err
	}

	// Determine the thread timestamp for the reply
	threadTs := event.ThreadTimeStamp
	if threadTs == "" {
		threadTs = event.TimeStamp // Use message timestamp if not in a thread
	}

	// Post the message in the same thread using thread_ts
	_, _, err = s.client.PostMessage(
		event.Channel,
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionTS(threadTs), // Ensure reply is in the thread
	)
	if err != nil {
		return &SlackError{Err: err, Context: "posting message"}
	}
	return nil
}

// processCommand handles different bot commands
func (s *SlackApp) processCommand(text string, userAD *User) (slack.Attachment, error) {
	attachment := slack.Attachment{}

	switch {
	case strings.Contains(text, "new"):
		if err := handleNewBooking(text, userAD, &attachment); err != nil {
			return attachment, err
		}
	case strings.Contains(text, "show"):
		if err := handleShowBookings(userAD, &attachment); err != nil {
			return attachment, err
		}
	case strings.Contains(text, "cancel"):
		if err := handleCancelBooking(text, userAD, &attachment); err != nil {
			return attachment, err
		}
	default:
		attachment.Text = getHelpMessage()
		attachment.Color = "#007bff"
	}
	return attachment, nil
}

func handleNewBooking(text string, userAD *User, attachment *slack.Attachment) error {
	words := strings.Fields(text)
	if len(words) < 8 {
		attachment.Text = "*Invalid format!*\nUse: `new <space> <date in DD-MM-YYY> <start in HH:MM> <date in DD-MM-YYY> <end in HH:MM>`\n\n" + getHelpMessage()
		attachment.Color = "#C00000"
		return nil
	}

	spaceName := strings.Title(words[2]) + " " + words[3]
	space, err := GetSpaceRepository().GetByKeyword(userAD.OrganizationID, spaceName)
	if err != nil || len(space) == 0 {
		attachment.Text = fmt.Sprintf("*Space* `%s` *not found.*", spaceName)
		attachment.Color = "#C00000"
		return nil
	}

	// Convert DD-MM-YYYY HH:MM to YYYY-MM-DD HH:MM:00
	enter, err := time.Parse("02-01-2006 15:04", words[4]+" "+words[5])
	if err != nil || enter.Before(time.Now()) {
		attachment.Text = "*Invalid or past start time.* Use: `DD-MM-YYYY HH:MM`"
		attachment.Color = "#C00000"
		return nil
	}
	enter = enter.Truncate(time.Minute)

	leave, err := time.Parse("02-01-2006 15:04", words[6]+" "+words[7])
	if err != nil || leave.Before(enter) {
		attachment.Text = "*Invalid end time or end is before start.* Use: `DD-MM-YYYY HH:MM`"
		attachment.Color = "#C00000"
		return nil
	}
	leave = leave.Truncate(time.Minute)

	booking := &Booking{
		SpaceID: space[0].ID,
		UserID:  userAD.ID,
		Enter:   enter,
		Leave:   leave,
	}

	conflicts, err := GetBookingRepository().GetConflicts(space[0].ID, enter, leave, "")
	if err != nil || len(conflicts) > 0 {
		attachment.Text = "*Requested desk is already occupied.*"
		attachment.Color = "#C00000"
		return nil
	}

	maxConcurrent, _ := GetSettingsRepository().GetInt(userAD.OrganizationID, SettingMaxConcurrentBookingsPerUser.Name)
	curAtTime, _ := GetBookingRepository().GetTimeRangeByUser(userAD.ID, enter, leave, "")
	if len(curAtTime) >= maxConcurrent {
		attachment.Text = "*You cannot have more than 1 concurrent booking.*\nTry again later."
		attachment.Color = "#C00000"
		return nil
	}

	if err := GetBookingRepository().Create(booking); err != nil {
		attachment.Text = "*Booking failed due to system error.*\nPlease report to the dev team."
		attachment.Color = "#C00000"
		return &SlackError{Err: err, Context: "creating booking"}
	}

	attachment.Text = fmt.Sprintf(
		"*Booking Confirmed!*\n\n*Space:* %s\n*From:* %s\n*To:* %s",
		space[0].Name,
		enter.Format("02-01-2006 15:04"),
		leave.Format("02-01-2006 15:04"),
	)
	attachment.Color = "#4af030"
	return nil
}

func handleShowBookings(userAD *User, attachment *slack.Attachment) error {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allBookings, err := GetBookingRepository().GetAllByUser(userAD.ID, time.Now().UTC())
	if err != nil {
		return &SlackError{
			Err:     err,
			Context: "fetching bookings from repository",
		}
	}

	if len(allBookings) == 0 {
		attachment.Text = "No bookings found"
		attachment.Color = "#C00000" // redColour
		return nil
	}

	attachment.Text = "*Your current & upcoming bookings:*\n"
	if err := formatOptions(allBookings, attachment); err != nil {
		return &SlackError{
			Err:     err,
			Context: "formatting booking options",
		}
	}
	return nil
}

// handleCancelBooking processes booking cancellation requests
func handleCancelBooking(text string, userAD *User, attachment *slack.Attachment) error {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bookings, err := GetBookingRepository().GetAllByUser(userAD.ID, time.Now().UTC())
	if err != nil {
		return &SlackError{
			Err:     err,
			Context: "fetching bookings for cancellation",
		}
	}

	if len(bookings) == 0 {
		attachment.Text = "*No bookings exist to cancel.*"
		attachment.Color = "#C00000"
		return nil
	}

	parts := strings.Fields(text)
	if len(parts) != 3 {
		attachment.Text = "*To cancel a booking, use:* `cancel <number>`\n\n"
		attachment.Color = "#C00000"
		return formatOptions(bookings, attachment)
	}

	number, err := strconv.Atoi(parts[2])
	if err != nil {
		attachment.Text = "*_Please provide a valid number next to cancel keyword. For reference check below_*\n\n" + getHelpMessage()
		attachment.Color = "#C00000" // redColour
		return nil
	}

	if number < 1 || number > len(bookings) {
		attachment.Text = "*Invalid booking number. Please choose from the list below:*\n"
		return formatOptions(bookings, attachment)
	}

	bookingToCancel := bookings[number-1]
	if err := GetBookingRepository().Delete(bookingToCancel); err != nil {
		return &SlackError{
			Err:     err,
			Context: "deleting booking",
		}
	}

	attachment.Text = fmt.Sprintf("*Booking #%d cancelled successfully*", number)
	attachment.Color = "#4af030"
	return nil
}

// formatOptions formats booking details for Slack message
func formatOptions(bookings []*BookingDetails, attachment *slack.Attachment) error {
	if len(bookings) == 0 {
		attachment.Text += "No bookings to display"
		attachment.Color = "#C00000" // redColour
		return nil
	}

	var message strings.Builder
	for i, booking := range bookings {
		enter, err := GetLocationRepository().AttachTimezoneInformation(booking.Enter, &booking.Space.Location)
		if err != nil {
			return &SlackError{
				Err:     err,
				Context: fmt.Sprintf("formatting enter time for booking %d", i+1),
			}
		}

		leave, err := GetLocationRepository().AttachTimezoneInformation(booking.Leave, &booking.Space.Location)
		if err != nil {
			return &SlackError{
				Err:     err,
				Context: fmt.Sprintf("formatting leave time for booking %d", i+1),
			}
		}

		message.WriteString(fmt.Sprintf(
			"*%d)* `%s - %s`\n• *From:* %s\n• *To:* %s\n\n",
			i+1,
			booking.Space.Location.Name,
			booking.Space.Name,
			enter.Format("02-01-2006 15:04"),
			leave.Format("02-01-2006 15:04"),
		))
	}

	attachment.Text += message.String()
	attachment.Color = "#4af030" // green
	return nil
}

// Shutdown gracefully terminates the Slack connection
func (s *SlackApp) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	// Wait for the event loop to finish
	s.wg.Wait()

	s.socketClient = nil

	if s.client != nil {
		s.client = nil
	}

	log.Println("SlackApp shutdown completed")
}

func getHelpMessage() string {
	return "*Available Commands:*\n" +
		"• `new Desk 2 01-05-2025 09:00 01-05-2025 17:00` – _Make a new booking_\n" +
		"• `show` – _View your current & upcoming bookings_\n" +
		"• `cancel <number>` – _Cancel a booking by its number_\n"
}

func getOrgID(searchName string) (string, error) {
	orgs, err := GetOrganizationRepository().GetAll()
	if err != nil {
		return "", err
	}
	for _, org := range orgs {
		if strings.ToLower(org.Name) == strings.ToLower(searchName) {
			return org.ID, nil
		}
	}
	return "", fmt.Errorf("Org Not Found")
}
