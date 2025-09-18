import React from 'react';

export const MyComponent: React.FC = () => {
  const reactInternals = React.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED;

  return (
    <div>
      <h1>Test Component</h1>
    </div>
  );
};