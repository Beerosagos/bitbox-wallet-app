/**
 * Copyright 2018 Shift Devices AG
 * Copyright 2022 Shift Crypto AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { QWebChannel } from './qwebchannel';
import { TMsgCallback, TQueryPromiseMap } from './transport-common';
import { runningInQtWebEngine } from './env';

let cache: any = null;
let webChannel: any = null;
let queryID: number = 0;
const queryPromises: TQueryPromiseMap = {};
const currentListeners: TMsgCallback[] = [];

function initTransport() {
  return new Promise((resolve, reject) => {
    if (webChannel) {
      resolve(webChannel);
    } else if (cache) {
      const check = function() {
        if (webChannel) {
          resolve(webChannel);
        } else {
          window.setTimeout(check, 1);
        }
      };
      check();
    } else if (runningInQtWebEngine()) {
      const initWebChannel = function(channel: any) {
        webChannel = channel;
        webChannel.objects.backend.gotResponse.connect((
          queryID: number,
          response: string,
        ) => {
          queryPromises[queryID].resolve(JSON.parse(response));
          delete queryPromises[queryID];
        });
        webChannel.objects.backend.pushNotify.connect((msg: string) => {
          currentListeners.forEach(listener => listener(JSON.parse(msg)));
        });
        resolve(webChannel);

      };
      cache = new QWebChannel((window.qt!).webChannelTransport, initWebChannel);
    } else {
      reject();
    }
  });
}

export function call(query: string) {
  return new Promise((resolve, reject) => {
    initTransport().then((channel: any) => {
      queryID++;
      queryPromises[queryID] = { resolve, reject };
      channel.objects.backend.call(queryID, query);
    });
  });
}

export function qtSubscribePushNotifications(msgCallback: TMsgCallback) {
  currentListeners.push(msgCallback);
  return () => {
    if (!currentListeners.includes(msgCallback)) {
      console.warn('!currentListeners.includes(msgCallback)');
    }
    const index = currentListeners.indexOf(msgCallback);
    currentListeners.splice(index, 1);
    if (currentListeners.includes(msgCallback)) {
      console.warn('currentListeners.includes(msgCallback)');
    }
  };
}