package chatter

import "context"

func (a *App) onUserConnect(ctx context.Context, username string) {
	// if user is not already connected, send a message to all friends that user is online
	friends, err := a.chatStore.GetFriends(ctx, username)
	if err != nil {
		return
	}
	payload := OnlineEventPayload{Username: username}
	a.eventRouter.EmitTo(OnlineEvent, payload, friends...)

}

func (a *App) onConnectionOpen(ctx context.Context, username string, i int) {

	friends, err := a.chatStore.GetFriends(ctx, username)
	if err != nil {
		return
	}
	// now send the online status of all friends to the user
	for _, friend := range friends {
		connected := a.wsManager.IsUserConnected(friend)
		if connected {
			payload := OnlineEventPayload{Username: friend}
			a.eventRouter.EmitTo(OnlineEvent, payload, username)
		}
	}
}

func (a *App) onUserDisconnect(ctx context.Context, username string) {
	friends, err := a.chatStore.GetFriends(ctx, username)
	if err != nil {
		return
	}
	payload := OfflineEventPayload{Username: username}
	a.eventRouter.EmitTo(OfflineEvent, payload, friends...)
}
