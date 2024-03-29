/*
 * Copyright 2018 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import * as React from 'react';
import BusyButton from '../../atoms/BusyButton';
import Button from '@material-ui/core/Button';
import Viewer, { ViewerConfig } from './Viewer';
import { Apis } from '../../lib/Apis';
import { commonCss, padding, color } from '../../Css';
import InputLabel from '@material-ui/core/InputLabel';
import Input from '@material-ui/core/Input';
import MenuItem from '@material-ui/core/MenuItem';
import ListSubheader from '@material-ui/core/ListSubheader';
import FormControl from '@material-ui/core/FormControl';
import Select from '@material-ui/core/Select';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogContentText from '@material-ui/core/DialogContentText';
import DialogTitle from '@material-ui/core/DialogTitle';
import { classes, stylesheet } from 'typestyle';

export const css = stylesheet({
  button: {
    marginBottom: 20,
    width: 150,
  },
  formControl: {
    minWidth: 120,
  },
  select: {
    minHeight: 50,
  },
  shortButton: {
    width: 50,
  },
  warningText: {
    color: color.warningText,
  },
  errorText: {
    color: color.errorText,
  },
});

export interface TensorboardViewerConfig extends ViewerConfig {
  url: string;
  namespace: string;
  podTemplateSpec?: any; // JSON object of pod template spec
  image?: string;
}

interface TensorboardViewerProps {
  configs: TensorboardViewerConfig[];
  // Interval in ms. If not specified, default to 5000.
  intervalOfCheckingTensorboardPodStatus?: number;
}

interface TensorboardViewerState {
  busy: boolean;
  deleteDialogOpen: boolean;
  podAddress: string;
  tfImage: string;
  // When podAddress is not null, we need to further tell whether the TensorBoard pod is accessible or not
  tensorboardReady: boolean;
  errorMessage?: string;
}

// TODO: bump default version when https://github.com/kubeflow/pipelines/issues/5521
// is resolved.
const DEFAULT_TF_IMAGE = 'tensorflow/tensorflow:2.2.2';

class TensorboardViewer extends Viewer<TensorboardViewerProps, TensorboardViewerState> {
  timerID: NodeJS.Timeout;

  constructor(props: any) {
    super(props);

    this.state = {
      busy: false,
      deleteDialogOpen: false,
      podAddress: '',
      tfImage: this._image() || DEFAULT_TF_IMAGE,
      tensorboardReady: false,
      errorMessage: undefined,
    };
  }

  public getDisplayName(): string {
    return 'Tensorboard';
  }

  public isAggregatable(): boolean {
    return true;
  }

  public componentDidMount(): void {
    this._checkTensorboardApp();
    this.timerID = setInterval(
      () => this._checkTensorboardPodStatus(),
      this.props.intervalOfCheckingTensorboardPodStatus || 5000,
    );
  }

  public componentWillUnmount(): void {
    clearInterval(this.timerID);
  }

  public handleImageSelect = (e: React.ChangeEvent<{ name?: string; value: unknown }>): void => {
    if (typeof e.target.value !== 'string') {
      throw new Error('Invalid event value type, expected string');
    }
    this.setState({ tfImage: e.target.value });
  };

  public render(): JSX.Element {
    return (
      <div>
        {this.state.errorMessage && <div className={css.errorText}>{this.state.errorMessage}</div>}
        {this.state.podAddress && (
          <div>
            <div
              className={padding(20, 'b')}
            >{`Tensorboard ${this.state.tfImage} is running for this output.`}</div>
            <a
              href={makeProxyUrl(this.state.podAddress)}
              target='_blank'
              rel='noopener noreferrer'
              className={commonCss.unstyled}
            >
              <Button
                className={classes(commonCss.buttonAction, css.button)}
                disabled={this.state.busy}
                color={'primary'}
              >
                Open Tensorboard
              </Button>
              {this.state.tensorboardReady ? (
                ``
              ) : (
                <div className={css.warningText}>
                  Tensorboard is starting, and you may need to wait for a few minutes.
                </div>
              )}
            </a>

            <div>
              <Button
                className={css.button}
                disabled={this.state.busy}
                id={'delete'}
                title={`stop tensorboard and delete its instance`}
                onClick={this._handleDeleteOpen}
                color={'default'}
              >
                Stop Tensorboard
              </Button>
              <Dialog
                open={this.state.deleteDialogOpen}
                onClose={this._handleDeleteClose}
                aria-labelledby='dialog-title'
              >
                <DialogTitle id='dialog-title'>{`Stop Tensorboard?`}</DialogTitle>
                <DialogContent>
                  <DialogContentText>
                    You can stop the current running tensorboard. The tensorboard viewer will also
                    be deleted from your workloads.
                  </DialogContentText>
                </DialogContent>
                <DialogActions>
                  <Button
                    className={css.shortButton}
                    id={'cancel'}
                    autoFocus={true}
                    onClick={this._handleDeleteClose}
                    color='primary'
                  >
                    Cancel
                  </Button>
                  <BusyButton
                    className={classes(commonCss.buttonAction, css.shortButton)}
                    onClick={this._deleteTensorboard}
                    busy={this.state.busy}
                    color='primary'
                    title={`Stop`}
                  />
                </DialogActions>
              </Dialog>
            </div>
          </div>
        )}

        {!this.state.podAddress && (
          <div>
            <div className={padding(30, 'b')}>
              <FormControl className={css.formControl}>
                <InputLabel htmlFor='viewer-tb-image-select'>TF Image</InputLabel>
                <Select
                  className={css.select}
                  value={this.state.tfImage}
                  input={<Input id='viewer-tb-image-select' />}
                  onChange={this.handleImageSelect}
                >
                  {this._image() && <MenuItem value={this._image()}>{this._image()}</MenuItem>}
                  <ListSubheader>Tensoflow 1.x</ListSubheader>
                  <MenuItem value={'tensorflow/tensorflow:1.7.1'}>TensorFlow 1.7.1</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.8.0'}>TensorFlow 1.8.0</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.9.0'}>TensorFlow 1.9.0</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.10.1'}>TensorFlow 1.10.1</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.11.0'}>TensorFlow 1.11.0</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.12.3'}>TensorFlow 1.12.3</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.13.2'}>TensorFlow 1.13.2</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.14.0'}>TensorFlow 1.14.0</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:1.15.5'}>TensorFlow 1.15.5</MenuItem>
                  <ListSubheader>TensorFlow 2.x</ListSubheader>
                  <MenuItem value={'tensorflow/tensorflow:2.0.4'}>TensorFlow 2.0.4</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:2.1.2'}>TensorFlow 2.1.2</MenuItem>
                  <MenuItem value={'tensorflow/tensorflow:2.2.2'}>TensorFlow 2.2.2</MenuItem>
                </Select>
              </FormControl>
            </div>
            <div>
              <BusyButton
                className={commonCss.buttonAction}
                disabled={!this.state.tfImage}
                onClick={this._startTensorboard}
                busy={this.state.busy}
                title={`Start ${this.props.configs.length > 1 ? 'Combined ' : ''}Tensorboard`}
              />
            </div>
          </div>
        )}
      </div>
    );
  }

  private _handleDeleteOpen = () => {
    this.setState({ deleteDialogOpen: true });
  };

  private _handleDeleteClose = () => {
    this.setState({ deleteDialogOpen: false });
  };

  private _getNamespace(): string {
    // TODO: We should probably check if all configs have the same namespace.
    return this.props.configs[0]?.namespace || '';
  }

  private _buildUrl(): string {
    const urls = this.props.configs.map(c => c.url).sort();
    return urls.length === 1 ? urls[0] : urls.map((c, i) => `Series${i + 1}:` + c).join(',');
  }

  private _podTemplateSpec(): any | undefined {
    const podTemplateSpec = this.props.configs[0]?.podTemplateSpec;
    // TODO: how to handle multiple config with different pod template specs?
    return podTemplateSpec || undefined;
  }

  private _image(): string | undefined {
    return this.props.configs[0]?.image || undefined;
  }

  private async _checkTensorboardPodStatus(): Promise<void> {
    // If pod address is not null and tensorboard pod doesn't seem to be read, pull status again
    if (this.state.podAddress && !this.state.tensorboardReady) {
      // Remove protocol prefix bofore ":" from pod address if any.
      Apis.isTensorboardPodReady(makeProxyUrl(this.state.podAddress)).then(ready => {
        this.setState(({ tensorboardReady }) => ({ tensorboardReady: tensorboardReady || ready }));
      });
    }
  }

  private async _checkTensorboardApp(): Promise<void> {
    this.setState({ busy: true }, async () => {
      try {
        // TODO: parse tfImage here
        const { podAddress, image } = await Apis.getTensorboardApp(
          this._buildUrl(),
          this._getNamespace(),
        );
        if (podAddress) {
          this.setState({ busy: false, podAddress, tfImage: image });
        } else {
          // No existing pod
          this.setState({ busy: false });
        }
      } catch (err) {
        this.setState({ busy: false, errorMessage: err?.message || 'Unknown error' });
      }
    });
  }

  private _startTensorboard = async () => {
    this.setState({ busy: true, errorMessage: undefined }, async () => {
      try {
        await Apis.startTensorboardApp({
          logdir: this._buildUrl(),
          namespace: this._getNamespace(),
          image: this.state.tfImage,
          podTemplateSpec: this._podTemplateSpec(),
        });
        this.setState({ busy: false, tensorboardReady: false }, () => {
          this._checkTensorboardApp();
        });
      } catch (err) {
        this.setState({ busy: false, errorMessage: err?.message || 'Unknown error' });
      }
    });
  };

  private _deleteTensorboard = async () => {
    // delete the already opened Tensorboard, clear the podAddress recorded in frontend,
    // and return to the select & start tensorboard page
    this.setState({ busy: true, errorMessage: undefined }, async () => {
      try {
        await Apis.deleteTensorboardApp(this._buildUrl(), this._getNamespace());
        this.setState({
          busy: false,
          deleteDialogOpen: false,
          podAddress: '',
          tensorboardReady: false,
        });
      } catch (err) {
        this.setState({ busy: false, errorMessage: err?.message || 'Unknown error' });
      }
    });
  };
}

function makeProxyUrl(podAddress: string) {
  // Strip the protocol from the URL. This is a workaround for cloud shell
  // incorrectly decoding the address and replacing the protocol's // with /.
  // Pod address (after stripping protocol) is of the format
  // <viewer_service_dns>.kubeflow.svc.cluster.local:6006/tensorboard/<viewer_name>/
  // We use this pod address without encoding since encoded pod address failed to open the
  // tensorboard instance on this pod.
  // TODO: figure out why the encoded pod address failed to open the tensorboard.
  return 'apis/v1/_proxy/' + podAddress.replace(/(^\w+:|^)\/\//, '');
}

export default TensorboardViewer;
