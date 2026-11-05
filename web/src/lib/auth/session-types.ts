import type { TokenPair } from "@/lib/api/types";

/** Public part of a token pair: what the browser is allowed to hold. */
export interface AccessSession {
  access_token: string;
  token_type: TokenPair["token_type"];
  expires_in: number;
}
