/**
 * Copyright 2023-2024 Shift Crypto AG
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

import { RefObject, useEffect, useRef, useState } from 'react';
import QrScanner from 'qr-scanner';
import { useTranslation } from 'react-i18next';

type TUseQRScannerOptions = {
  onStart?: () => void;
  onResult: (result: QrScanner.ScanResult) => void;
  onError: (error: any) => void;
}

export const useQRScanner = (
  videoRef: RefObject<HTMLVideoElement>, {
    onStart,
    onResult,
    onError,
  }: TUseQRScannerOptions
) => {
  const { t } = useTranslation();
  const [initErrorMessage, setInitErrorMessage] = useState();
  const scanner = useRef<QrScanner | null>(null);
  const wip = useRef<boolean>(false);

  useEffect(() => {

    console.log('render');

    (async () => {
      if (!videoRef.current) {
        return;
      }

      while (wip.current) {
        await new Promise(r => setTimeout(r, 100));
      }
      console.log('create');
      try {
        wip.current = true;
        console.log('wip: ' + wip.current);
        const randomId = Math.random().toString();
        videoRef.current.id = randomId;
        scanner.current = new QrScanner(
          videoRef.current,
          result => {
            console.log('result');
            scanner.current?.stop();
            onResult(result);
          }, {
            onDecodeError: err => {
              console.log('decode output: ' + videoRef.current?.id);

              const errorString = err.toString();
              if (err && !errorString.includes('No QR code found')) {
                console.log('decode error');
                onError(err);
              }
            },
            highlightScanRegion: true,
            highlightCodeOutline: true,
            calculateScanRegion: (v) => {
              const videoWidth = v.videoWidth;
              const videoHeight = v.videoHeight;
              const factor = 0.5;
              const size = Math.floor(Math.min(videoWidth, videoHeight) * factor);
              return {
                x: (videoWidth - size) / 2,
                y: (videoHeight - size) / 2,
                width: size,
                height: size
              };
            }
          }
        );
        scanner.current.$video.id = randomId;
        videoRef.current.onchange = () => console.log('change');
        await new Promise(r => setTimeout(r, 300));
        console.log('starting ' + scanner.current.$video.id + '...');
        await scanner.current?.start();
        console.log('started');

        wip.current = false;
        console.log('wip: ' + wip.current);
        console.log('all done');
        if (!videoRef.current) {
          console.log('videoRef expired');
        }
        if (onStart) {
          onStart();
        }
      } catch (error: any) {
        const stringifiedError = error.toString();
        console.log('start failed: ' + stringifiedError);
        wip.current = false;
        console.log('wip: ' + wip.current);
        const cameraNotFound = stringifiedError === 'Camera not found.';
        setInitErrorMessage(cameraNotFound ? t('send.scanQRNoCameraMessage') : stringifiedError);
        onError(error);
      }

    })();

    return () => {
      (async() => {
        console.log('unmounting');
        while (wip.current) {
          await new Promise(r => setTimeout(r, 100));
        }
        if (scanner.current) {
          wip.current = true;
          console.log('wip: ' + wip.current);
          console.log('Still running. Stopping' + scanner.current.$video.id + '...');
          await scanner.current?.pause(true);
          await scanner.current?.stop();
          console.log('stopped. destroying...');
          await scanner.current?.destroy();
          console.log('destroyed');
          scanner.current = null;
          wip.current = false;
          console.log('wip: ' + wip.current);
          console.log('all done');

        }

      })();
    };
  }, [videoRef, onStart, onResult, onError, t]);

  return { initErrorMessage };
};
