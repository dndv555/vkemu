package store

const schema = `
CREATE TABLE IF NOT EXISTS counters (
    name  TEXT PRIMARY KEY,
    value INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    uid             INTEGER PRIMARY KEY,
    first_name      TEXT NOT NULL,
    last_name       TEXT NOT NULL DEFAULT '',
    nickname        TEXT NOT NULL DEFAULT '',
    domain          TEXT NOT NULL DEFAULT '',
    sex             INTEGER NOT NULL DEFAULT 0,
    bdate           TEXT NOT NULL DEFAULT '',
    city            INTEGER NOT NULL DEFAULT 0,
    country         INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT '',
    mobile_phone    TEXT NOT NULL DEFAULT '',
    home_phone      TEXT NOT NULL DEFAULT '',
    university_name TEXT NOT NULL DEFAULT '',
    graduation      TEXT NOT NULL DEFAULT '',
    relation        INTEGER NOT NULL DEFAULT 0,
    online          INTEGER NOT NULL DEFAULT 0,
    password        TEXT NOT NULL DEFAULT '',
    avatar          TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sessions (
    sid     TEXT PRIMARY KEY,
    uid     INTEGER NOT NULL,
    secret  TEXT NOT NULL,
    created INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS friends (
    uid INTEGER NOT NULL,
    fid INTEGER NOT NULL,
    PRIMARY KEY (uid, fid)
);

CREATE TABLE IF NOT EXISTS friend_requests (
    from_uid INTEGER NOT NULL,
    to_uid   INTEGER NOT NULL,
    message  TEXT NOT NULL DEFAULT '',
    date     INTEGER NOT NULL,
    PRIMARY KEY (from_uid, to_uid)
);

CREATE TABLE IF NOT EXISTS wall_posts (
    owner_id      INTEGER NOT NULL,
    post_id       INTEGER NOT NULL,
    from_id       INTEGER NOT NULL,
    date          INTEGER NOT NULL,
    text          TEXT NOT NULL DEFAULT '',
    likes         INTEGER NOT NULL DEFAULT 0,
    comments      INTEGER NOT NULL DEFAULT 0,
    copy_owner_id INTEGER NOT NULL DEFAULT 0,
    copy_post_id  INTEGER NOT NULL DEFAULT 0,
    attach_type   TEXT NOT NULL DEFAULT '',
    attach_json   TEXT NOT NULL DEFAULT '',
    geo_json      TEXT NOT NULL DEFAULT '',
    deleted       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner_id, post_id)
);

CREATE TABLE IF NOT EXISTS comments (
    ctype        TEXT NOT NULL,
    owner_id     INTEGER NOT NULL,
    item_id      INTEGER NOT NULL,
    cid          INTEGER NOT NULL,
    from_id      INTEGER NOT NULL,
    date         INTEGER NOT NULL,
    text         TEXT NOT NULL DEFAULT '',
    reply_to_uid INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (ctype, owner_id, item_id, cid)
);

CREATE TABLE IF NOT EXISTS likes (
    ctype    TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    item_id  INTEGER NOT NULL,
    uid      INTEGER NOT NULL,
    PRIMARY KEY (ctype, owner_id, item_id, uid)
);

CREATE TABLE IF NOT EXISTS messages (
    mid         INTEGER PRIMARY KEY,
    from_id     INTEGER NOT NULL,
    to_id       INTEGER NOT NULL,
    chat_id     INTEGER NOT NULL DEFAULT 0,
    date        INTEGER NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    read_state  INTEGER NOT NULL DEFAULT 0,
    attach_json TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS audio (
    owner_id INTEGER NOT NULL,
    aid      INTEGER NOT NULL,
    album_id INTEGER NOT NULL DEFAULT 0,
    artist   TEXT NOT NULL DEFAULT '',
    title    TEXT NOT NULL DEFAULT '',
    duration INTEGER NOT NULL DEFAULT 0,
    url      TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (owner_id, aid)
);

CREATE TABLE IF NOT EXISTS audio_albums (
    owner_id INTEGER NOT NULL,
    album_id INTEGER NOT NULL,
    title    TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (owner_id, album_id)
);

CREATE TABLE IF NOT EXISTS albums (
    owner_id    INTEGER NOT NULL,
    aid         INTEGER NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    privacy     INTEGER NOT NULL DEFAULT 0,
    thumb_id    INTEGER NOT NULL DEFAULT -1,
    created     INTEGER NOT NULL,
    PRIMARY KEY (owner_id, aid)
);

CREATE TABLE IF NOT EXISTS photos (
    owner_id INTEGER NOT NULL,
    pid      INTEGER NOT NULL,
    aid      INTEGER NOT NULL DEFAULT 0,
    date     INTEGER NOT NULL,
    text     TEXT NOT NULL DEFAULT '',
    src      TEXT NOT NULL DEFAULT '',
    src_big  TEXT NOT NULL DEFAULT '',
    likes    INTEGER NOT NULL DEFAULT 0,
    comments INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner_id, pid)
);

CREATE TABLE IF NOT EXISTS videos (
    owner_id    INTEGER NOT NULL,
    vid         INTEGER NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    duration    INTEGER NOT NULL DEFAULT 0,
    image       TEXT NOT NULL DEFAULT '',
    url         TEXT NOT NULL DEFAULT '',
    date        INTEGER NOT NULL,
    PRIMARY KEY (owner_id, vid)
);

CREATE TABLE IF NOT EXISTS notes (
    owner_id INTEGER NOT NULL,
    nid      INTEGER NOT NULL,
    title    TEXT NOT NULL DEFAULT '',
    text     TEXT NOT NULL DEFAULT '',
    date     INTEGER NOT NULL,
    comments INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner_id, nid)
);

CREATE TABLE IF NOT EXISTS docs (
    owner_id INTEGER NOT NULL,
    did      INTEGER NOT NULL,
    title    TEXT NOT NULL DEFAULT '',
    ext      TEXT NOT NULL DEFAULT '',
    size     INTEGER NOT NULL DEFAULT 0,
    url      TEXT NOT NULL DEFAULT '',
    date     INTEGER NOT NULL,
    PRIMARY KEY (owner_id, did)
);

CREATE TABLE IF NOT EXISTS groups (
    gid       INTEGER PRIMARY KEY,
    name      TEXT NOT NULL,
    is_closed INTEGER NOT NULL DEFAULT 0,
    is_admin  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS polls (
    owner_id INTEGER NOT NULL,
    poll_id  INTEGER NOT NULL,
    question TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (owner_id, poll_id)
);

CREATE TABLE IF NOT EXISTS poll_answers (
    owner_id  INTEGER NOT NULL,
    poll_id   INTEGER NOT NULL,
    answer_id INTEGER NOT NULL,
    text      TEXT NOT NULL DEFAULT '',
    votes     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner_id, poll_id, answer_id)
);

CREATE TABLE IF NOT EXISTS poll_votes (
    owner_id  INTEGER NOT NULL,
    poll_id   INTEGER NOT NULL,
    uid       INTEGER NOT NULL,
    answer_id INTEGER NOT NULL,
    PRIMARY KEY (owner_id, poll_id, uid)
);

CREATE TABLE IF NOT EXISTS cities (
    cid  INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS countries (
    cid  INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS places (
    place_id  INTEGER PRIMARY KEY,
    title     TEXT NOT NULL,
    address   TEXT NOT NULL DEFAULT '',
    latitude  REAL NOT NULL DEFAULT 0,
    longitude REAL NOT NULL DEFAULT 0,
    type      INTEGER NOT NULL DEFAULT 1,
    checkins  INTEGER NOT NULL DEFAULT 0,
    city_id   INTEGER NOT NULL DEFAULT 0,
    country_id INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS checkins (
    id        INTEGER PRIMARY KEY,
    uid       INTEGER NOT NULL,
    place_id  INTEGER NOT NULL,
    date      INTEGER NOT NULL,
    text      TEXT NOT NULL DEFAULT '',
    latitude  REAL NOT NULL DEFAULT 0,
    longitude REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS uploads (
    hash     TEXT PRIMARY KEY,
    kind     TEXT NOT NULL,
    owner_id INTEGER NOT NULL DEFAULT 0,
    path     TEXT NOT NULL,
    size     INTEGER NOT NULL DEFAULT 0,
    title    TEXT NOT NULL DEFAULT '',
    created  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wall_posts_date ON wall_posts(owner_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date);
CREATE INDEX IF NOT EXISTS idx_comments_item ON comments(ctype, owner_id, item_id, cid);
`
