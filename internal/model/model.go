package model

type User struct {
	UID            int
	FirstName      string
	LastName       string
	Nickname       string
	Domain         string
	Sex            int
	BDate          string
	City           int
	Country        int
	Status         string
	MobilePhone    string
	HomePhone      string
	UniversityName string
	Graduation     string
	Relation       int
	Online         bool
	Password       string
	Avatar         string
}

func (u User) FullName() string {
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

type Session struct {
	SID     string
	UID     int
	Secret  string
	Created int64
}

type FriendRequest struct {
	FromUID int
	ToUID   int
	Message string
	Date    int64
}

type WallPost struct {
	OwnerID     int
	PostID      int
	FromID      int
	Date        int64
	Text        string
	Likes       int
	Comments    int
	CopyOwnerID int
	CopyPostID  int
	AttachType  string
	AttachJSON  string
	GeoJSON     string
	Deleted     bool
}

type Comment struct {
	CType      string
	OwnerID    int
	ItemID     int
	CID        int
	FromID     int
	Date       int64
	Text       string
	ReplyToUID int
}

type Message struct {
	MID        int
	FromID     int
	ToID       int
	ChatID     int
	Date       int64
	Body       string
	ReadState  bool
	AttachJSON string
}

func (m Message) PeerFor(uid int) int {
	if m.FromID == uid {
		return m.ToID
	}
	return m.FromID
}

func (m Message) OutFor(uid int) bool {
	return m.FromID == uid
}

type Audio struct {
	AID      int
	OwnerID  int
	AlbumID  int
	Artist   string
	Title    string
	Duration int
	URL      string
}

type AudioAlbum struct {
	AlbumID int
	OwnerID int
	Title   string
}

type Album struct {
	AID         int
	OwnerID     int
	Title       string
	Description string
	Privacy     int
	ThumbID     int
	Created     int64
}

type Photo struct {
	PID      int
	OwnerID  int
	AID      int
	Date     int64
	Text     string
	Src      string
	SrcBig   string
	Likes    int
	Comments int
}

type Video struct {
	VID         int
	OwnerID     int
	Title       string
	Description string
	Duration    int
	Image       string
	URL         string
	Date        int64
}

type Note struct {
	NID      int
	OwnerID  int
	Title    string
	Text     string
	Date     int64
	Comments int
}

type Doc struct {
	DID     int
	OwnerID int
	Title   string
	Ext     string
	Size    int
	URL     string
	Date    int64
}

type Group struct {
	GID      int
	Name     string
	IsClosed bool
	IsAdmin  bool
}

type PollAnswer struct {
	ID    int    `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

type Poll struct {
	PollID   int
	OwnerID  int
	Question string
	Answers  []PollAnswer
}

type City struct {
	CID  int
	Name string
}

type Country struct {
	CID  int
	Name string
}

type Place struct {
	PlaceID   int
	Title     string
	Address   string
	Latitude  float64
	Longitude float64
	Type      int
	Checkins  int
	CityID    int
	CountryID int
}

type Checkin struct {
	ID        int
	UID       int
	PlaceID   int
	Date      int64
	Text      string
	Latitude  float64
	Longitude float64
}

type Upload struct {
	Hash    string
	Kind    string
	OwnerID int
	Path    string
	Size    int
	Title   string
	Created int64
}
