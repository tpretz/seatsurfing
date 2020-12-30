import React from 'react';
import FullLayout from '../components/FullLayout';
import Loading from '../components/Loading';
import { User, Organization, AuthProvider, Settings as OrgSettings, Domain, Ajax } from 'flexspace-commons';
import { Form, Col, Row, Table, Button, Alert, InputGroup, Popover, OverlayTrigger } from 'react-bootstrap';
import { Link, Redirect } from 'react-router-dom';
import { Plus as IconPlus, Save as IconSave } from 'react-feather';
import { withTranslation } from 'react-i18next';
import { TFunction } from 'i18next';

interface State {
  allowAnyUser: boolean
  confluenceClientId: string
  maxBookingsPerUser: number
  maxDaysInAdvance: number
  maxBookingDurationHours: number
  subscriptionActive: boolean
  subscriptionMaxUsers: number
  selectedAuthProvider: string
  loading: boolean
  submitting: boolean
  saved: boolean
  error: boolean
  newDomain: string
  domains: Domain[]
  userDomain: string
}

interface Props {
  t: TFunction
}

class Settings extends React.Component<Props, State> {
  org: Organization | null;
  authProviders: AuthProvider[];

  constructor(props: any) {
    super(props);
    this.org = null;
    this.authProviders = [];
    this.state = {
      allowAnyUser: true,
      confluenceClientId: "",
      maxBookingsPerUser: 0,
      maxBookingDurationHours: 0,
      maxDaysInAdvance: 0,
      subscriptionActive: false,
      subscriptionMaxUsers: 0,
      selectedAuthProvider: "",
      loading: true,
      submitting: false,
      saved: false,
      error: false,
      newDomain: "",
      domains: [],
      userDomain: ""
    };
  }

  componentDidMount = () => {
    this.loadSettings();
    this.loadItems();
  }

  loadItems = () => {
    User.getSelf().then(user => {
      let userDomain = user.email.substring(user.email.indexOf("@")+1).toLowerCase();
      Organization.get(user.organizationId).then(org => {
        this.org = org;
        Domain.list(org.id).then(domains => {
          this.setState({
            domains: domains,
            userDomain: userDomain
          });
          AuthProvider.list().then(list => {
            this.authProviders = list;
            this.setState({ loading: false });
          });
        });
      });
    });
  }

  loadSettings = () => {
    OrgSettings.list().then(settings => {
      let state: any = {};
      settings.forEach(s => {
        if (s.name === "allow_any_user") state.allowAnyUser = (s.value === "1");
        if (s.name === "confluence_client_id") state.confluenceClientId = s.value;
        if (s.name === "max_bookings_per_user") state.maxBookingsPerUser = window.parseInt(s.value);
        if (s.name === "max_days_in_advance") state.maxDaysInAdvance = window.parseInt(s.value);
        if (s.name === "max_booking_duration_hours") state.maxBookingDurationHours = window.parseInt(s.value);
        if (s.name === "subscription_active") state.subscriptionActive = (s.value === "1");
        if (s.name === "subscription_max_users") state.subscriptionMaxUsers = window.parseInt(s.value);
      });
      this.setState({
        ...this.state,
        ...state
      });
    });
  }

  onSubmit = (e: any) => {
    e.preventDefault();
    this.setState({
      submitting: true,
      saved: false,
      error: false
    });
    let payload = [
      new OrgSettings("allow_any_user", this.state.allowAnyUser ? "1" : "0"),
      new OrgSettings("confluence_client_id", this.state.confluenceClientId),
      new OrgSettings("max_bookings_per_user", this.state.maxBookingsPerUser.toString()),
      new OrgSettings("max_days_in_advance", this.state.maxDaysInAdvance.toString()),
      new OrgSettings("max_booking_duration_hours", this.state.maxBookingDurationHours.toString())
    ];
    OrgSettings.setAll(payload).then(() => {
      this.setState({
        submitting: false,
        saved: true
      });
    }).catch(() => {
      this.setState({
        submitting: false,
        error: true
      });
    });
  }

  onAuthProviderSelect = (e: AuthProvider) => {
    this.setState({ selectedAuthProvider: e.id });
  }

  getAuthProviderTypeLabel = (providerType: number): string => {
    switch (providerType) {
      case 1: return "OAuth 2";
      default: return "Unknown";
    }
  }

  renderAuthProviderItem = (e: AuthProvider) => {
    return (
      <tr key={e.id} onClick={() => this.onAuthProviderSelect(e)}>
        <td>{e.name}</td>
        <td>{this.getAuthProviderTypeLabel(e.providerType)}</td>
      </tr>
    );
  }

  verifyDomain = (domainName: string) => {
    document.body.click();
    this.state.domains.forEach(domain => {
      if (domain.domain === domainName) {
        domain.verify().then(() => {
          Domain.list(domain.organizationId).then(domains => this.setState({ domains: domains }));
        }).catch(e => {
          alert(this.props.t("errorValidateDomain", {domain: domainName}));
        })
      }
    });
  }

  isValidDomain = () => {
    if (this.state.newDomain.indexOf(".") < 3) {
      return false;
    }
    let lastIndex = this.state.newDomain.length - 3;
    if (lastIndex < 3) {
      lastIndex = 3;
    }
    if (this.state.newDomain.lastIndexOf(".") > lastIndex) {
      return false;
    }
    return true;
  }

  addDomain = () => {
    if (!this.isValidDomain() || !this.org) {
      return;
    }
    Domain.add(this.org.id, this.state.newDomain).then(() => {
      Domain.list(this.org ? this.org.id : "").then(domains => this.setState({ domains: domains }));
      this.setState({ newDomain: "" });
    }).catch(() => {
      alert(this.props.t("errorAddDomain"));
    });
  }

  removeDomain = (domainName: string) => {
    if (!window.confirm(this.props.t("confirmDeleteDomain", {domain: domainName}))) {
      return;
    }
    this.state.domains.forEach(domain => {
      if (domain.domain === domainName) {
        domain.delete().then(() => {
          Domain.list(this.org ? this.org.id : "").then(domains => this.setState({ domains: domains }));
        }).catch(() => alert(this.props.t("errorDeleteDomain")));
      }
    });
  }

  handleNewDomainKeyDown = (target: any) => {
    if (target.key === "Enter") {
      target.preventDefault();
      this.addDomain();
    }
  }

  deleteOrg = () => {
    if (window.confirm(this.props.t("confirmDeleteOrg"))) {
      if (window.confirm(this.props.t("confirmDeleteOrg2"))) {
        this.org?.delete().then(() => {
          Ajax.JWT = "";
          window.sessionStorage.removeItem("jwt");
          window.location.href = "/admin/";
        });
      }
    }
  }

  manageSubscription = () => {
    let windowRef = window.open();
    this.org?.getSubscriptionManagementURL().then(url => {
      if (windowRef) {
        windowRef.location.href = url;
      }
    }).catch(() => {
      if (windowRef) {
        windowRef?.close();
      }
      alert(this.props.t("errorTryAgain"));
    });
  }

  render() {
    if (this.state.selectedAuthProvider) {
      return <Redirect to={`/settings/auth-providers/${this.state.selectedAuthProvider}`} />
    }

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("settings")}>
          <Loading />
        </FullLayout>
      );
    }

    let domains = this.state.domains.map(domain => {
      let verify = <></>;
      let popoverId = "popover-domain-" + domain.domain;
      const popover = (
        <Popover id={popoverId}>
          <Popover.Title as="h3">{this.props.t("verifyDomain")}</Popover.Title>
          <Popover.Content>
            <div>{this.props.t("verifyDomainHowto", {domain: domain.domain})}</div>
            <div>&nbsp;</div>
            <div><strong>seatsurfing-verification={domain.verifyToken}</strong></div>
            <div>&nbsp;</div>
            <Button variant="primary" size="sm" onClick={() => this.verifyDomain(domain.domain)}>{this.props.t("verifyNow")}</Button>
          </Popover.Content>
        </Popover>
      );
      if (!domain.active) {
        verify = (
          <OverlayTrigger trigger="click" placement="auto" overlay={popover} rootClose={false}>
            <Button variant="primary" size="sm">{this.props.t("verify")}</Button>
          </OverlayTrigger>
        );
      }
      let key = "domain-" + domain.domain;
      let canDelete = domain.domain.toLowerCase() !== this.state.userDomain;
      return (
        <Form.Group key={key}>
          {domain.domain}
            &nbsp;
          <Button variant="danger" size="sm" onClick={() => this.removeDomain(domain.domain)} disabled={!canDelete}>{this.props.t("remove")}</Button>
            &nbsp;
          {verify}
        </Form.Group>
      );
    });

    let authProviderRows = this.authProviders.map(item => this.renderAuthProviderItem(item));
    let authProviderTable = <p>{this.props.t("noRecords")}</p>;
    if (authProviderRows.length > 0) {
      authProviderTable = (
        <Table striped={true} hover={true} className="clickable-table">
          <thead>
            <tr>
              <th>{this.props.t("name")}</th>
              <th>{this.props.t("type")}</th>
            </tr>
          </thead>
          <tbody>
            {authProviderRows}
          </tbody>
        </Table>
      );
    }

    let subscription = <></>;
    if (this.state.subscriptionActive) {
      subscription = (
        <>
          <p>{this.props.t("subscriptionActive", {num: this.state.subscriptionMaxUsers})}</p>
          <p><Button variant="primary" onClick={this.manageSubscription}>{this.props.t("subscriptionManage")}</Button></p>
        </>
      );
    } else {
      subscription = (
        <>
          <p>{this.props.t("subscriptionInactive", {num: this.state.subscriptionMaxUsers})}</p>
          <p><Button variant="primary" onClick={this.manageSubscription}>{this.props.t("subscriptionManage")}</Button></p>
        </>
      );
    }

    let dangerZone = (
      <>
        <Button className="btn btn-danger" onClick={this.deleteOrg}>{this.props.t("deleteOrg")}</Button>
      </>
    );

    let hint = <></>;
    if (this.state.saved) {
      hint = <Alert variant="success">{this.props.t("entryUpdated")}</Alert>
    } else if (this.state.error) {
      hint = <Alert variant="danger">{this.props.t("errorSave")}</Alert>
    }

    let buttonSave = <Button className="btn-sm" variant="outline-secondary" type="submit" form="form"><IconSave className="feather" /> {this.props.t("save")}</Button>;
    let contactName = "";
    if (this.org) {
      contactName = this.org.contactFirstname + " " + this.org.contactLastname + " ("+this.org.contactEmail+")";
    }

    return (
      <FullLayout headline={this.props.t("settings")} buttons={buttonSave}>
        <Form onSubmit={this.onSubmit} id="form">
          {hint}
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("org")}</Form.Label>
            <Col sm="4">
              <Form.Control plaintext={true} readOnly={true} defaultValue={this.org?.name} />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("primaryContact")}</Form.Label>
            <Col sm="4">
              <Form.Control plaintext={true} readOnly={true} defaultValue={contactName} />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Col sm="6">
              <Form.Check type="checkbox" id="check-allowAnyUser" label={this.props.t("allowAnyUser")} checked={this.state.allowAnyUser} onChange={(e: any) => this.setState({ allowAnyUser: e.target.checked })} />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("maxBookingsPerUser")}</Form.Label>
            <Col sm="4">
              <Form.Control type="number" value={this.state.maxBookingsPerUser} onChange={(e: any) => this.setState({ maxBookingsPerUser: e.target.value })} min="1" max="9999" />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("maxDaysInAdvance")}</Form.Label>
            <Col sm="4">
              <InputGroup>
                <Form.Control type="number" value={this.state.maxDaysInAdvance} onChange={(e: any) => this.setState({ maxDaysInAdvance: e.target.value })} min="0" max="9999" />
                <InputGroup.Append>
                  <InputGroup.Text>{this.props.t("days")}</InputGroup.Text>
                </InputGroup.Append>
              </InputGroup>
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("maxBookingDurationHours")}</Form.Label>
            <Col sm="4">
              <InputGroup>
                <Form.Control type="number" value={this.state.maxBookingDurationHours} onChange={(e: any) => this.setState({ maxBookingDurationHours: e.target.value })} min="0" max="9999" />
                <InputGroup.Append>
                  <InputGroup.Text>{this.props.t("hours")}</InputGroup.Text>
                </InputGroup.Append>
              </InputGroup>
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("confluenceClientId")}</Form.Label>
            <Col sm="4">
              <Form.Control type="text" value={this.state.confluenceClientId} onChange={(e: any) => this.setState({ confluenceClientId: e.target.value })} />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">{this.props.t("domains")}</Form.Label>
            <Col sm="4">
              {domains}
              <InputGroup size="sm">
                <Form.Control type="text" value={this.state.newDomain} onChange={(e: any) => this.setState({ newDomain: e.target.value })} placeholder="ihre-domain.de" onKeyDown={this.handleNewDomainKeyDown} />
                <InputGroup.Append>
                  <Button variant="outline-secondary" onClick={this.addDomain} disabled={!this.isValidDomain()}>{this.props.t("addDomain")}</Button>
                </InputGroup.Append>
              </InputGroup>
            </Col>
          </Form.Group>
        </Form>
        <div className="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
          <h1 className="h2">{this.props.t("subscription")}</h1>
        </div>
        {subscription}
        <div className="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
          <h1 className="h2">{this.props.t("authProviders")}</h1>
          <div className="btn-toolbar mb-2 mb-md-0">
            <div className="btn-group mr-2">
              <Link to="/settings/auth-providers/add" className="btn btn-sm btn-outline-secondary"><IconPlus className="feather" /> {this.props.t("add")}</Link>
            </div>
          </div>
        </div>
        {authProviderTable}
        <div className="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
          <h1 className="h2">{this.props.t("dangerZone")}</h1>
        </div>
        {dangerZone}
      </FullLayout>
    );
  }
}

export default withTranslation()(Settings as any);
