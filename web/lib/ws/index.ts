import { decodeEvent, WSEvent, encodeEvent } from "@/lib/ws/event";

type WSOptions = {
  onStateChange: (readyState: ReadyState) => void;
  handlers: Record<string, EventHandler>;
};

export enum ReadyState {
  Connecting = "connecting",
  Open = "connected",
  Closing = "disconnecting",
  Closed = "disconnected",
}

export type EventHandler = (e: WSEvent) => void;

export class WS {
  private conn!: WebSocket;
  readyState: ReadyState = ReadyState.Closed;
  private onReadyStateChange: (readyState: ReadyState) => void;
  private handlers: Record<string, EventHandler> = {};

  constructor({ onStateChange, handlers }: WSOptions) {
    this.onReadyStateChange = onStateChange;
    this.handlers = handlers;
  }

  connect() {
    if (this.readyState !== ReadyState.Closed) {
      return;
    }
    this.conn = new WebSocket("ws://localhost:8080/ws");
    this.readyState = ReadyState.Connecting;

    this.conn.addEventListener("open", () => {
      console.log("Connection opened");
      this.setReadyState(ReadyState.Open);
    });

    this.conn.addEventListener("message", (message) => {
      const packet = decodeEvent(message.data);
      if (packet === null) {
        console.error("Cannot decode event", message.data);
        return;
      }
      const handler = this.handlers[packet.type];
      if (!handler) {
        console.error("No handler for packet type", packet);
        return;
      }
      handler(packet);
    });

    this.conn.addEventListener("close", (ev) => {
      console.log("Connection closed", ev);
      this.setReadyState(ReadyState.Closed);
    });

    this.conn.addEventListener("error", (error) => {
      console.log("Error", error);
      this.setReadyState(ReadyState.Closing);
      this.conn.close();
    });
  }

  setReadyState(readyState: ReadyState) {
    this.readyState = readyState;
    this.onReadyStateChange(this.readyState);
  }

  sendPacket(packet: WSEvent) {
    if (this.readyState !== ReadyState.Open) {
      return;
    }
    this.conn.send(encodeEvent(packet));
  }

  public close() {
    this.conn.close();
  }
}
