type Level = 'info' | 'warn' | 'error' | 'debug';

export function createLogger(component: string) {
  return {
    info: (msg: string, data?: object) =>
      console.info(`[${component}]`, msg, data ?? ''),
    warn: (msg: string, data?: object) =>
      console.warn(`[${component}]`, msg, data ?? ''),
    error: (msg: string, data?: object) =>
      console.error(`[${component}]`, msg, data ?? ''),
    debug: (msg: string, data?: object) => {
      if (process.env.NODE_ENV === 'development')
        console.debug(`[${component}]`, msg, data ?? '');
    },
  };
}
