export type SolveRequest = {
  url: string;
  type: string;
  profile_id: string;
  cdp_endpoint?: string;
  kameleo_base?: string;
  enable_script?: boolean;
  enable_token?: boolean;
  enable_provider?: boolean;
};

export type SolveResponse = {
  success: boolean;
  message: string;
  tier?: string;
};

export type TierResult = {
  ok: boolean;
  tier: string;
  message: string;
};

export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
