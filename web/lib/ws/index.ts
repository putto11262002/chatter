import { z } from "zod";

type WSOptions = {
  onStateChange: (readyState: ReadyState) => void;
  onPacketReceived: (packet: Packet) => void;
};

type Packet = {
  type: string;
  id?: number;
  body?: unknown;
};

export function createPacket(type: string, body: unknown): Packet {
  return {
    type,
    body,
    id: Math.floor(Math.random() * 65536),
  };
}

const PacketSchema = z.object({
  type: z.string(),
  body: z.unknown(),
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
  Connecting = 0,
  Open = 1,
  Closing = 2,
  Closed = 3,
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
    this.conn = new WebSocket("ws://localhost:8080/ws/?id=1234");
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
    this.conn.send(encodePacket(packet));
  }

  public close() {
    this.conn.close();
  }
}
