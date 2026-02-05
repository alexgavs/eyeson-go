import type { SessionData } from '../types/app';
import { SESSION_DURATION_MS, STORAGE_KEYS } from '../constants/app';

export class SessionManager {
  static save(token: string, username: string): void {
    const session: SessionData = {
      token,
      username,
      expiresAt: Date.now() + SESSION_DURATION_MS
    };
    localStorage.setItem(STORAGE_KEYS.session, JSON.stringify(session));
  }

  static load(): SessionData | null {
    try {
      const data = localStorage.getItem(STORAGE_KEYS.session);
      if (!data) return null;

      const session: SessionData = JSON.parse(data);
      if (Date.now() > session.expiresAt) {
        this.clear();
        return null;
      }
      return session;
    } catch {
      return null;
    }
  }

  static clear(): void {
    localStorage.removeItem(STORAGE_KEYS.session);
  }

  static getToken(): string | null {
    const session = this.load();
    return session?.token || null;
  }
}
