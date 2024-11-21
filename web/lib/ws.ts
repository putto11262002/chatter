import { Message } from "@/types/chat";
import { z } from "zod";

type WSOptions = {
  onReadStateChange: (readyState: ReadyState) => void;
  onPacketReceived: (packet: Packet) => void;
};

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

  constructor({ onReadStateChange, onPacketReceived }: WSOptions) {
    this.onReadyStateChange = onReadStateChange;
    this.onPacketReceived = onPacketReceived;
  }

  connect() {
    if (this.readyState !== ReadyState.Closed) {
      return;
    }
    this.conn = new WebSocket("ws://localhost:8080/api/ws");
    this.readyState = ReadyState.Connecting;

    this.conn.addEventListener("open", () => {
      console.log("Connection opened");
      this.setReadyState(ReadyState.Open);
    });

    this.conn.addEventListener("message", (message) => {
      const messageValidation = PacketSchema.safeParse(
        JSON.parse(message.data)
      );
      if (messageValidation.success) {
        const packet = messageValidation.data;
        packet.data = base64Decode(packet.data);

        this.onPacketReceived(packet);
        return;
      }
      console.log(
        "invalid packet received",
        "\ndata:",
        message.data,
        "\nerror:",
        messageValidation.error.format()
      );
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
    this.conn.send(JSON.stringify(packet));
  }

  public close() {
    this.conn.close();
  }
}

export enum PacketType {
  ChatMessage = 1,
  RoomEvent = 2,
  Error = 3,
  ChatMessageStatusUpdate = 4,
  ReadRoomMessages = 5,
  TypingEvent = 6,
}

export type Packet = {
  type: PacketType;
  // base64 encoded string
  data: string;
  correlationID: number;
};

export const PacketSchema = z.object({
  type: z.nativeEnum(PacketType),
  data: z.string(),
  correlationID: z.number(),
});

export function createPacket(type: PacketType, data: string): Packet {
  return {
    correlationID: generateCorrelationID(),
    type,
    data: base64Encode(data),
  };
}

export function createChatMessagePacket(
  message: Pick<Message, "data" | "type" | "roomID">
): Packet {
  return createPacket(PacketType.ChatMessage, JSON.stringify(message));
}

function base64Decode(str: string): string {
  return atob(str);
}

function base64Encode(str: string): string {
  return btoa(str);
}

function generateCorrelationID(): number {
  return Math.floor(Math.random() * 65536);
}
