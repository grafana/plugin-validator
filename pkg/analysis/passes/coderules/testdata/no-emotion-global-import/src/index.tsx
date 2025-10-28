import React from 'react';
import { Global } from '@emotion/react';

export const App = () => (
  <>
    <Global styles={{ body: { margin: 0 } }} />
    <div>Hello</div>
  </>
);

const Layout = () => (
  <>
    <Global />
    <main>Content</main>
  </>
);
