/**
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

import { translate, TranslateProps } from '../../decorators/translate';
import { useState, useEffect, FunctionComponent, createRef } from 'react';
import { MessageVersion, parseMessage, serializeMessage, V0MessageType } from 'request-address';
import { signAddr, getPocketURL } from '../../api/backend';
import { Header } from '../../components/layout';
import { Spinner } from '../../components/spinner/Spinner';
import { useLoad } from '../../hooks/api';
import Guide from './guide';
import style from './iframe.module.css';

interface PocketProps {
    code: string;
}

type Props = PocketProps & TranslateProps;

// TODO:
// - add T&C
const PocketComponent: FunctionComponent<Props> = ({ code, t }) => {
  const [height, setHeight] = useState(0);
  const iframeURL = useLoad(getPocketURL(code));

  const ref = createRef<HTMLDivElement>();
  const iframeRef = createRef<HTMLIFrameElement>();
  var resizeTimerID: any = undefined;

  const name = 'Bitcoin';

  useEffect(() => {
    onResize();
    window.addEventListener('resize', onResize);
    window.addEventListener('message', onMessage);

    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('message', onMessage);
    };
  });

  const onResize = () => {
    if (resizeTimerID) {
      clearTimeout(resizeTimerID);
    }
    resizeTimerID = setTimeout(() => {
      if (!ref.current) {
        return;
      }
      setHeight(ref.current.offsetHeight);
    }, 200);
  };

  const onMessage = (e: MessageEvent) => {
    try {
      const message = parseMessage(e.data);
      console.log(JSON.stringify(message, undefined, 2));

      if (!code) {
        return;
      }

      // handle address request from moonpay
      if (message.type === V0MessageType.RequestAddress && message.withMessageSignature) {
        signAddr(
          message.withScriptType ? message.withScriptType : 'p2wpkh',
          String(message.withMessageSignature),
          code)
          .then( resp => {
            console.log(JSON.stringify(resp));

            if (!resp.addr || !resp.sig ) {
              console.log('Something went wrong!');
            } else {
              sendAddress(resp.addr, resp.sig);
            }
          });
      }
    } catch (e) {
      // ignore messages that could not be parsed
      // probably not intended for us, anyway
    }
  };

  const sendAddress = (address: string, sig: string) => {
    const { current } = iframeRef;

    if (!current) {
      return;
    }

    const message = serializeMessage({
      version: MessageVersion.V0,
      type: V0MessageType.Address,
      bitcoinAddress: address,
      signature: sig,
    });

    console.log(message);
    current.contentWindow?.postMessage(message, '*');
  };

  return (
    <div className="contentWithGuide">
      <div className="container">
        <div className={style.header}>
          <Header title={<h2>{t('buy.info.title', { name })}</h2>} />
        </div>
        <div ref={ref} className="innerContainer">
          <div className="noSpace" style={{ height }}>
            <Spinner text={t('loading')} />
            <iframe
              ref={iframeRef}
              title="Pocket"
              width="100%"
              height={height}
              frameBorder="0"
              className={style.iframe}
              allow="camera; payment"
              src={iframeURL}>
            </iframe>
          </div>
        </div>
      </div>
      <Guide name={name} />
    </div>
  );
};

export const Pocket = translate()(PocketComponent);
