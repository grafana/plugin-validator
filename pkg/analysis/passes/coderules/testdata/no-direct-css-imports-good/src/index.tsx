import React from 'react';
import styled from 'styled-components';
import { css } from '@emotion/react';

const GlobalStyles = styled.div`
  margin: 0;
  padding: 0;
`;

const styles = css`
  body {
    margin: 0;
  }
`;

export const App = () => (
  <GlobalStyles>
    <div>Hello World</div>
  </GlobalStyles>
);
