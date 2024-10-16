/**
 * Copyright 2018 Shift Devices AG
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

import React, { useState } from 'react';
import { testRegister } from '../../../api/account';
import { getTesting } from '../../../api/backend';
import { Button } from '../../../components/forms';
import { PasswordSingleInput } from '../../../components/password';
import { Dialog, DialogButtons } from '../../../components/dialog/dialog';
import { useLoad } from '../../../hooks/api';
import { debug } from '../../../utils/env';

export const SkipForTesting = () => {
  const [dialog, setDialog] = useState(false);
  const show = useLoad(debug ? getTesting : () => Promise.resolve(false));
  const [testPIN, setTestPIN] = useState('');
  const registerTestingDevice = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    await testRegister(testPIN);
    setDialog(false);
  };

  if (!show) {
    return null;
  }
  const title = 'Unlock software kestore';
  return (
    <>
      <button onClick={() => setDialog(true)}>{title}</button>
      <Dialog open={dialog} title={title} onClose={() => setDialog(false)}>
        <form onSubmit={registerTestingDevice}>
          <PasswordSingleInput
            type="password"
            autoFocus
            label="Test Password"
            onValidPassword={setTestPIN}
            value={testPIN} />
          <DialogButtons>
            <Button primary type="submit">
              Unlock
            </Button>
          </DialogButtons>
        </form>
      </Dialog>
    </>
  );
};
