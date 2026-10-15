import { Code, ConnectError, createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { WheelService } from "@/gen/wheel/v1/wheel_pb";
import { auth } from "./firebase";

const authInterceptor: Interceptor = (next) => async (req) => {
  const user = auth.currentUser;
  if (user) {
    req.header.set("Authorization", `Bearer ${await user.getIdToken()}`);
  }
  return next(req);
};

export const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || "http://localhost:7755",
  interceptors: [authInterceptor],
});

export const wheelClient = createClient(WheelService, transport);

/** Turns any thrown value into a message fit for a snackbar. */
export function friendlyError(err: unknown): string {
  const e = ConnectError.from(err);
  switch (e.code) {
    case Code.Unauthenticated:
      return "Please sign in first.";
    case Code.PermissionDenied:
      return "Only the wheel's owner can do that.";
    case Code.NotFound:
      return "That wheel doesn't exist (any more).";
    case Code.InvalidArgument:
    case Code.FailedPrecondition:
      return e.rawMessage || "That isn't possible right now.";
    case Code.Unavailable:
    case Code.Unknown:
      return "Can't reach the wheel server. Is it running?";
    default:
      return e.rawMessage || "Something went wrong.";
  }
}
