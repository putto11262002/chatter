export type Room = {
  id: string;
  users: RoomUser[];
  type: string;
};

export type RoomUser = {
  username: string;
  roomID: string;
  roomName: string;
};

export type CreatePrivateChatRequest = {
  other: string;
};

export type CreatePrivateChatResponse = {
  id: string;
};

export enum MessageType {
  TEXT = 1,
}

export type MessageCreateRequest = {
  data: string;
  type: MessageType;
};

export type Message = {
  data: string;
  sender: string;
  type: MessageType;
  sendTime: string;
};
