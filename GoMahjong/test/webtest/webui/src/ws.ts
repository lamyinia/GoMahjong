// WebSocket wrapper with reconnection support

import type { WSMessage, LogMessage } from './types';

export type MessageHandler = (msg: WSMessage) => void;
export type LogHandler = (log: LogMessage) => void;
export type ConnectionHandler = (connected: boolean) => void;

export class WSClient {
  private ws: WebSocket | null = null;
  private url: string;
  private playerId: string | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  
  private onMessage: MessageHandler | null = null;
  private onLog: LogHandler | null = null;
  private onConnectionChange: ConnectionHandler | null = null;

  constructor() {
    // Determine WebSocket URL based on current location
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    this.url = `${protocol}//${location.host}/ws`;
  }

  setHandlers(
    onMessage: MessageHandler,
    onLog: LogHandler,
    onConnection: ConnectionHandler
  ) {
    this.onMessage = onMessage;
    this.onLog = onLog;
    this.onConnectionChange = onConnection;
  }

  connect(playerId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      this.playerId = playerId;
      
      try {
        this.ws = new WebSocket(this.url);
      } catch (err) {
        reject(err);
        return;
      }

      this.ws.onopen = () => {
        console.log('[WS] Connected');
        this.reconnectAttempts = 0;
        
        // Send init message with playerId
        this.ws?.send(JSON.stringify({ playerId }));
        
        this.onConnectionChange?.(true);
        resolve();
      };

      this.ws.onclose = () => {
        console.log('[WS] Disconnected');
        this.onConnectionChange?.(false);
        
        // Attempt reconnect
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          console.log(`[WS] Reconnecting in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts})`);
          setTimeout(() => {
            if (this.playerId) {
              this.connect(this.playerId);
            }
          }, this.reconnectDelay);
        }
      };

      this.ws.onerror = (err) => {
        console.error('[WS] Error:', err);
        reject(err);
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          // Check if it's a log message
          if ('level' in data && 'message' in data) {
            this.onLog?.(data as LogMessage);
          } else if ('route' in data) {
            this.onMessage?.(data as WSMessage);
          }
        } catch (err) {
          console.error('[WS] Parse error:', err);
        }
      };
    });
  }

  disconnect() {
    this.playerId = null;
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnect
    this.ws?.close();
    this.ws = null;
  }

  send(route: string, payload: unknown): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.error('[WS] Not connected');
      return false;
    }

    const msg: WSMessage = { route, payload };
    this.ws.send(JSON.stringify(msg));
    return true;
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
export const wsClient = new WSClient();
