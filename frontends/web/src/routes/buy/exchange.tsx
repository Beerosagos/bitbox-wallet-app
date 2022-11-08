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
import { FunctionComponent } from 'react';
import { Button } from '../../components/forms';
import { isPocketSupported, isMoonpayBuySupported } from '../../api/backend';
import { route } from '../../utils/route';
import { useLoad } from '../../hooks/api';

interface ExchangeProps {
    code: string;
}

type Props = ExchangeProps & TranslateProps;
// TODO:
// - add layout
const ExchangeComponent: FunctionComponent<Props> = ({ code, t }) => {
  const showPocket = useLoad(isPocketSupported(code));
  const showMoonpay = useLoad(isMoonpayBuySupported(code));

  const goToExchange = (exchange: string) => {
    route(`/buy/${exchange}/${code}`);
  };

  return (
    <div>
      <div>{t('Choose you exchange!')}</div>
      <div>
        { showMoonpay && (<Button
          primary
          onClick={() => goToExchange('moonpay')} >
          {t('Moonpay')}
        </Button>) }
      </div>
      <div>
        { showPocket && (<Button
          primary
          onClick={() => goToExchange('pocket')} >
          {t('Pocket')}
        </Button>) }
      </div>

    </div>
  );
};

export const Exchange = translate()(ExchangeComponent);
