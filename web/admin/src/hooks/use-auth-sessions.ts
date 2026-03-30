import { useSyncExternalStore } from "react";
import {
  getCurrentSession,
  getSessions,
  subscribeAuthChanges,
  type AuthSession,
} from "@/lib/auth";

interface AuthSessionsState {
  currentSession: AuthSession | null;
  sessions: AuthSession[];
}

function getSnapshot(): AuthSessionsState {
  return {
    currentSession: getCurrentSession(),
    sessions: getSessions(),
  };
}

export function useAuthSessions(): AuthSessionsState {
  return useSyncExternalStore(subscribeAuthChanges, getSnapshot, getSnapshot);
}
