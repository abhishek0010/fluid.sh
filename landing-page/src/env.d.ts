interface ImportMetaEnv {
  readonly PUBLIC_POSTHOG_API_KEY: string;
  readonly PUBLIC_POSTHOG_HOST: string;
  readonly PUBLIC_POSTHOG_DEFAULTS: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
