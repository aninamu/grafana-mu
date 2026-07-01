import { useEffect, useRef, useState } from 'react';
import * as React from 'react';
import { SkeletonTheme } from 'react-loading-skeleton';

import { type GrafanaTheme2, ThemeContext } from '@grafana/data';
import { ThemeChangedEvent, config } from '@grafana/runtime';

import { appEvents } from '../app_events';
import { initAccentFromUrl } from '../services/theme';

import 'react-loading-skeleton/dist/skeleton.css';

export const ThemeProvider = ({ children, value }: { children: React.ReactNode; value: GrafanaTheme2 }) => {
  const [theme, setTheme] = useState(value);
  const isInitialValue = useRef(true);

  useEffect(() => {
    const sub = appEvents.subscribe(ThemeChangedEvent, (event) => {
      config.theme2 = event.payload;
      setTheme(event.payload);
    });

    initAccentFromUrl();

    return () => sub.unsubscribe();
  }, []);

  useEffect(() => {
    if (isInitialValue.current) {
      isInitialValue.current = false;
      return;
    }
    setTheme(value);
  }, [value]);

  return (
    <ThemeContext.Provider value={theme}>
      <SkeletonTheme
        baseColor={theme.colors.emphasize(theme.colors.background.secondary)}
        highlightColor={theme.colors.emphasize(theme.colors.background.secondary, 0.1)}
        borderRadius={theme.shape.radius.default}
      >
        {children}
      </SkeletonTheme>
    </ThemeContext.Provider>
  );
};
