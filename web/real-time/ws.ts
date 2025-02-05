import { z } from "zod";

type WSOptions = {
  onStateChange: (readyState: ReadyState) => void;
  onPacketReceived: (packet: Packet) => void;
};

export type Packet = {
  type: string;
  id?: number;
  payload?: unknown;
};

export function createPacket(type: string, body: unknown): Packet {
  return {
    type,
    payload: body,
    id: Math.floor(Math.random() * 65536),
  };
}

const PacketSchema = z.object({
  type: z.string(),
  payload: z.unknown(),
});

function encodePacket(packet: Packet): string {
  return JSON.stringify(packet);
}

function decodePacket(raw: string): Packet | null {
  try {
    const parsed = JSON.parse(raw);
    const packet = PacketSchema.parse(parsed);
    return packet;
  } catch {
    return null;
  }
}

export enum ReadyState {
  Connecting = "connecting",
  Open = "connected",
  Closing = "disconnecting",
  Closed = "disconnected",
}

export class WS {
  private conn!: WebSocket;
  readyState: ReadyState = ReadyState.Closed;
  private onReadyStateChange: (readyState: ReadyState) => void;
  private onPacketReceived: (packet: Packet) => void;

  constructor({ onStateChange, onPacketReceived }: WSOptions) {
    this.onReadyStateChange = onStateChange;
    this.onPacketReceived = onPacketReceived;
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
      const packet = decodePacket(message.data);
      if (packet === null) {
        console.error("Invalid packet", message.data);
        return;
      }
      this.onPacketReceived(packet);
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

  sendPacket(packet: Packet) {
    if (this.readyState !== ReadyState.Open) {
      return;
    }
    this.conn.send(encodePacket(packet));
  }

  public close() {
    this.conn.close();
  }
}
