package api

func (s *Server) buildRoutes() map[string]handler {
	return map[string]handler{
		"execute": s.execute,

		"auth.getTokenSecure":   s.authGetToken,
		"auth.getToken":         s.authGetToken,
		"auth.getSessionSecure": s.authGetSession,
		"auth.getSession":       s.authGetSession,
		"auth.logout":           s.authLogout,
		"getViewerId":           s.getViewerID,

		"getProfiles":            s.usersGet,
		"users.get":              s.usersGet,
		"users.search":           s.usersSearch,
		"users.getSubscriptions": s.usersGetSubscriptions,

		"friends.get":         s.friendsGet,
		"friends.getOnline":   s.friendsGetOnline,
		"friends.getMutual":   s.friendsGetMutual,
		"friends.getRequests": s.friendsGetRequests,
		"friends.add":         s.friendsAdd,
		"friends.delete":      s.friendsDelete,
		"friends.getByPhones": s.friendsGetByPhones,
		"friends.areFriends":  s.friendsAreFriends,

		"wall.get":           s.wallGet,
		"wall.getById":       s.wallGetByID,
		"wall.post":          s.wallPost,
		"wall.delete":        s.wallDelete,
		"wall.getComments":   s.wallGetComments,
		"wall.addComment":    s.wallAddComment,
		"wall.deleteComment": s.wallDeleteComment,
		"wall.addLike":       s.wallAddLike,
		"wall.deleteLike":    s.wallDeleteLike,
		"wall.edit":          s.wallEdit,

		"messages.getDialogs":        s.messagesGetDialogs,
		"messages.getHistory":        s.messagesGetHistory,
		"messages.getById":           s.messagesGetByID,
		"messages.get":               s.messagesGet,
		"messages.send":              s.messagesSend,
		"messages.delete":            s.messagesDelete,
		"messages.markAsRead":        s.messagesMarkAsRead,
		"messages.search":            s.messagesSearch,
		"messages.getLongPollServer": s.messagesGetLongPollServer,

		"newsfeed.get":         s.newsfeedGet,
		"newsfeed.getComments": s.newsfeedGetComments,

		"photos.get":                 s.photosGet,
		"photos.getUserPhotos":       s.photosGetUserPhotos,
		"photos.getById":             s.photosGetByID,
		"photos.getAlbums":           s.photosGetAlbums,
		"photos.createAlbum":         s.photosCreateAlbum,
		"photos.editAlbum":           s.photosEditAlbum,
		"photos.deleteAlbum":         s.photosDeleteAlbum,
		"photos.delete":              s.photosDelete,
		"photos.edit":                s.photosEdit,
		"photos.createComment":       s.photosCreateComment,
		"photos.getComments":         s.photosGetComments,
		"photos.getTags":             s.photosGetTags,
		"photos.getUploadServer":     s.photosGetUploadServer,
		"photos.getWallUploadServer": s.photosGetWallUploadServer,
		"photos.save":                s.photosSave,
		"photos.saveWallPhoto":       s.photosSaveWallPhoto,

		"audio.get":                s.audioGet,
		"audio.getById":            s.audioGetByID,
		"audio.getRecommendations": s.audioGetRecommendations,
		"audio.search":             s.audioSearch,
		"audio.getAlbums":          s.audioGetAlbums,
		"audio.add":                s.audioAdd,
		"audio.delete":             s.audioDelete,
		"audio.getUploadServer":    s.audioGetUploadServer,
		"audio.save":               s.audioSave,

		"video.get":         s.videoGet,
		"video.save":        s.videoSave,
		"video.getComments": s.videoGetComments,

		"notes.getById":     s.notesGetByID,
		"notes.getComments": s.notesGetComments,

		"docs.getUploadServer": s.docsGetUploadServer,
		"docs.save":            s.docsSave,

		"groups.getById": s.groupsGetByID,
		"getGroupsFull":  s.groupsGetFull,

		"status.get": s.statusGet,
		"status.set": s.statusSet,

		"likes.add":    s.likesAdd,
		"likes.delete": s.likesDelete,

		"polls.getById": s.pollsGetByID,
		"polls.addVote": s.pollsAddVote,

		"places.search":         s.placesSearch,
		"places.getById":        s.placesGetByID,
		"places.add":            s.placesAdd,
		"places.checkin":        s.placesCheckin,
		"places.getCheckins":    s.placesGetCheckins,
		"places.getCityById":    s.placesGetCityByID,
		"places.getCountryById": s.placesGetCountryByID,

		"activity.online": s.activityOnline,
		"getCounters":     s.getCounters,

		"captcha.force": s.captchaForce,
	}
}
