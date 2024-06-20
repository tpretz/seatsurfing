import React from 'react';
import { Modal, Button } from 'react-bootstrap';
import { WithTranslation, withTranslation } from 'next-i18next';

interface State {
}

interface Props extends WithTranslation {
  heading: string,
  showModal: boolean,
  submitFunction(): void,
  submitButtonText: string,
  body: any,
  closeModal(): void
}

class GenericModal extends React.Component<Props, State> {
  constructor(props: any) {
    super(props);
    };

  render() {
    return (
      <>
        <Modal show={this.props.showModal} onHide={() => this.props.closeModal()}>
          <Modal.Header closeButton>
            <Modal.Title>{this.props.t(this.props.heading)}</Modal.Title>
          </Modal.Header>
          <Modal.Body>
            <p>{this.props.body}</p>
          </Modal.Body>
          <Modal.Footer>
            <Button variant="secondary" onClick={() => this.props.closeModal()}>
              {this.props.t("back")}
            </Button>
            <Button variant="danger" onClick={() => this.props.submitFunction()}>
              {this.props.t(this.props.submitButtonText)}
            </Button>
          </Modal.Footer>
        </Modal>
      </>
    );
  }
}

export default withTranslation()(GenericModal as any);