// Worker that fronts the Goalrail container and proxies all HTTP (and
// WebSocket) traffic to it. Goalrail needs a SINGLE server instance (in-memory
// runner registry), so every request routes to one fixed container instance.
import { Container, getContainer } from "@cloudflare/containers";

export class GoalrailServer extends Container {
  // Port the Goalrail server listens on inside the container.
  defaultPort = 8000;
  // Keep the container warm so D1-backed sessions don't cold-start constantly.
  sleepAfter = "30m";

  constructor(ctx, env) {
    super(ctx, env);
    // Env passed into the container. Secrets (DATABASE_URL, the cookie secret,
    // the AWS_* R2 keys) come from `wrangler secret put`; the rest are plain
    // vars in wrangler.jsonc.
    this.envVars = {
      DATABASE_URL: env.DATABASE_URL,
      GOALRAIL_ACCOUNTS_COOKIE_SECRET: env.GOALRAIL_ACCOUNTS_COOKIE_SECRET,
      GOALRAIL_AUTH_ENABLED: "1",
      GOALRAIL_AUTH_PROVIDER: "accounts",
      GOALRAIL_ACCOUNTS_AUTO_OPEN: "0",
      HOST: "0.0.0.0",
      PORT: "8000",
      // Artifact store -> R2 over the S3 API (goalrail's native S3 backend).
      // GOALRAIL_ARTIFACT_URI selects it; AWS_* point boto3 at R2.
      GOALRAIL_ARTIFACT_URI: env.GOALRAIL_ARTIFACT_URI,
      AWS_ENDPOINT_URL_S3: env.AWS_ENDPOINT_URL_S3,
      AWS_DEFAULT_REGION: "auto",
      AWS_ACCESS_KEY_ID: env.AWS_ACCESS_KEY_ID,
      AWS_SECRET_ACCESS_KEY: env.AWS_SECRET_ACCESS_KEY,
    };
  }
}

export default {
  async fetch(request, env) {
    // One shared instance for the whole app (single-replica requirement).
    return await getContainer(env.GOALRAIL, "singleton").fetch(request);
  },
};
