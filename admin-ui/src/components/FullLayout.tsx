import React, { JSX, ReactNode } from 'react';
import NavBar from './NavBar';
import SideBar from './SideBar';
import RuntimeConfig from './RuntimeConfig';

interface Props {
  headline: string
  buttons?: ReactNode
  children: string | JSX.Element | JSX.Element[]
}

interface State {
}

export default class FullLayout extends React.Component<Props, State> {
  render() {
    if (RuntimeConfig.INFOS.pluginWelcomeScreens && RuntimeConfig.INFOS.pluginWelcomeScreens.length > 0) {
      return (
        <div style={{ height: '100vh', width: '100vw', overflow: 'hidden' }}>
          <iframe src={RuntimeConfig.INFOS.pluginWelcomeScreens[0].src} style={{ width: '100%', height: '100vh', borderWidth: 0 }} id="welcome-screen-iframe">
          </iframe>
        </div>
      );
    }

    return (
      <div>
        <NavBar />
        <div className="container-fluid">
          <div className="row">
            <SideBar />
            <main role="main" className="col-md-9 ms-sm-auto col-lg-10 px-md-4">
              <div className="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
                <h1 className="h2">{this.props.headline}</h1>
                <div className="btn-toolbar mb-2 mb-md-0">
                  <div className="btn-group me-2">
                    {this.props.buttons}
                  </div>
                </div>
              </div>
              {this.props.children}
            </main>
          </div>
        </div>
      </div>
    );
  }
}
