![Go](https://img.shields.io/badge/Go-1.24-blue)
![SQLite](https://img.shields.io/badge/SQLite-3-green)
![DeepSeek](https://img.shields.io/badge/AI-DeepSeek%20V4.1%20Flash-orange)
![License](https://img.shields.io/badge/License-MIT-yellow)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
# vkemu

Эмулятор серверного API старого клиента VK для Android (`api_id=2274003`, API `3.0`,
`http://api.vkontakte.ru/api.php`) плюс простая веб-версия. Хранилище — SQLite,
HTTP — стандартная библиотека Go.

# Версия клиента
ВКонтакте 1.3.2.

# Информация
Проект никак не связан с компанией ООО «ВК».
Проект написал DeepSeek V4.1 Flash.

## Запуск

```
go run ./cmd/vkemu -addr :8080 -web-addr :80 -db data/vkemu.db
```

Поднимаются два сервера:

* `-addr` — API для Android-клиента (`/api.php`, `/longpoll`, `/upload/...`, `/media/...`).
* `-web-addr` — веб-версия на порту 80 (`/login`, `/register`, `/feed`, `/music`).

Флаги (или переменные окружения `VKEMU_*`):

| флаг | env | по умолчанию | смысл |
| --- | --- | --- | --- |
| `-addr` | `VKEMU_ADDR` | `:8080` | адрес API |
| `-web-addr` | `VKEMU_WEB_ADDR` | `:80` | адрес веб-версии |
| `-db` | `VKEMU_DB` | `data/vkemu.db` | файл SQLite |
| `-uploads` | `VKEMU_UPLOADS` | `data/uploads` | каталог загруженных файлов |
| `-base-url` | `VKEMU_BASE_URL` | из `-addr` | внешний адрес API, например `http://10.0.2.2:8080` для эмулятора Android |
| `-longpoll-host` | `VKEMU_LONGPOLL_HOST` | из `-base-url` | хост long poll в формате `host:port/longpoll` |
| `-require-signature` | `VKEMU_REQUIRE_SIGNATURE` | `false` | проверять `sig` и `digest` |
| `-app-id` | `VKEMU_APP_ID` | `2274003` | ожидаемый `api_id` |
| `-app-secret` | `VKEMU_APP_SECRET` | секрет клиента | секрет secure-авторизации |
| `-longpoll-wait` | `VKEMU_LONGPOLL_WAIT` | `25` | ожидание long poll, секунды |
| `-rate-limit` | `VKEMU_RATE_LIMIT` | `50` | запросов в секунду на один IP (`0` — выключить) |
| `-rate-burst` | `VKEMU_RATE_BURST` | `100` | максимальный всплеск запросов на один IP |
| `-max-conns` | `VKEMU_MAX_CONNS` | `512` | максимум одновременных обработок (`0` — без лимита) |
| `-trust-proxy` | `VKEMU_TRUST_PROXY` | `false` | брать IP из `X-Forwarded-For` |

База стартует пустой: пользователи, стена, музыка, фото и места создаются через веб-версию
или клиент. В справочники при старте добавляются только города и страны — никаких засеянных
мест-заглушек нет, список мест формируется отметками пользователей.

## Веб-версия

Порт 80, обычный HTML без CSS. Веб-версия повторяет почти все возможности API.

Лента и записи:

* `/feed` — новости как в клиенте: записи пользователя и друзей, постраничная выдача по
  `end_time`, форма новой записи с любым вложением (фото, аудио, видео, документ),
  лайк прямо в ленте.
* `/post/view?owner=&id=` — просмотр записи: автор, вложение, лайк, комментарии, удаление.
* `/post`, `/post/delete`, `/like`, `/comment` — создать, удалить, лайкнуть, прокомментировать.
* `/register`, `/login`, `/logout` — аккаунт и сессия в cookie `vkemu_web`.

Люди:

* `/id/<uid>` — профиль: аватар, статус, полная информация (день рождения, город, страна,
  учебное заведение, семейное положение, телефоны), стена, добавление/удаление из друзей,
  статистика.
* `/friends`, `/friends/requests`, `/friend/add`, `/friend/accept`, `/friend/delete`.
* `/search` — поиск людей по имени и логину.

Общение:

* `/messages` — диалоги и беседы, выбор друга из списка и переход в переписку;
  `/messages?uid=` — переписка, `/messages/send`, `/messages/delete`.
* Ссылка «Написать сообщение» есть на странице профиля друга и в списке друзей.

## Анти-DDoS

Встроенный ограничитель по IP с алгоритмом token bucket, общий для API и веб-версии
(`internal/ratelimit`). Запросы сверх лимита получают `429 Too Many Requests` с заголовком
`Retry-After`, при переполнении одновременных обработок — `503 Service Unavailable`.

* `-rate-limit` — устойчивая скорость запросов в секунду на один IP, `-rate-burst` — размер
  всплеска.
* `-max-conns` — предел одновременных обработок, защита от исчерпания ресурсов.
* `-trust-proxy` — учитывать `X-Forwarded-For`, если сервер стоит за обратным прокси.
* long poll (`/longpoll`) исключён из лимита: он висит до 25 секунд в ожидании обновлений.

Счётчики клиентов очищаются фоновым процессом, поэтому память не растёт.

Медиа:

* `/photos` — галерея и альбомы, `/photos/upload`, `/photos/delete`, `/albums/create`.
* `/music` — список треков и загрузка, `/music/upload`, `/music/delete`.
* `/videos`, `/videos/upload` — видео и воспроизведение.
* `/notes`, `/notes/add`, `/notes/delete` — заметки.
* `/docs`, `/docs/upload`, `/docs/delete` — документы.
* `/upload-avatar` — аватарка; подставляется в `photo`/`photo_rec`/`photo_big` API.

Прочее:

* `/places`, `/places/checkin` — места и отметки (создают запись с `geo`).
* `/settings` — редактирование профиля: имя, фамилия, никнейм, пол, дата рождения, город,
  страна, учебное заведение и год окончания, семейное положение, телефоны, статус;
  `/status` — быстрая смена статуса.
* `/media/...` — загруженные файлы, placeholder-картинки и тишина в WAV.

Медиа отдаётся с корректным `Content-Type` по расширению файла и поддерживает `Range`-запросы
(`206 Partial Content`), поэтому аудио и видео воспроизводятся клиентом и браузером. Ссылки на
загруженные файлы включают расширение (`/media/upload/<hash>.mp3`), иначе клиент не мог
положить файл в кэш по расширению и `MediaPlayer` не воспроизводил трек.

Ключевой момент для Android-клиента: все абсолютные ссылки (`photo`, `src`, `url`, `upload_url`,
`server` long poll) формируются из хоста того запроса, который пришёл от клиента, а не из
`-base-url`. Клиент жёстко обращается к `api.vkontakte.ru`; если хост в ссылках не совпадает с
тем, через который клиент реально достучался до эмулятора (например `10.0.2.2:8080` из
эмулятора Android), аватарка, фото и музыка не загрузятся и long poll не подключится.
`-base-url` используется только как запасной вариант, когда хост запроса недоступен.

Файлы WAV для заглушек-треков генерируются в 16-битном формате: 8-битный WAV `MediaPlayer`
на Android не декодирует, из-за чего встроенные треки не играли.

## API

Эндпоинты:

* `POST /api.php` — методы API и `execute`.
* `GET /longpoll?act=a_check&key=&ts=&wait=` — long poll.
* `POST /upload/...` — приём файлов (фото, аудио, видео, документы, аватары).
* `GET /media/...` — placeholder-медиа и загруженные файлы.
* `GET /captcha.png` — картинка капчи.

Чтобы направить клиент на эмулятор, подмените `api.vkontakte.ru` в `APIRequest`/`Auth`
на `-base-url` (либо проксируйте DNS).

## Структура

```
cmd/vkemu            точка входа, поднимает API и веб-версию
internal/config      конфигурация и флаги
internal/model       доменные структуры
internal/store       SQLite: схема, справочники, регистрация, доступ к данным по сущностям
internal/params      разбор параметров запроса, подпись, MD5
internal/vkscript    интерпретатор VKScript для метода execute
internal/longpoll    шина обновлений long poll
internal/media       генерация placeholder-медиа (PNG, WAV)
internal/upload      разбор multipart и сохранение загруженных файлов
internal/ratelimit   анти-DDoS: token bucket по IP и предел одновременных обработок
internal/api         маршрутизация, форматирование ответов, методы API
internal/web         веб-версия на чистом HTML
```

## Поддерживаемые методы

Авторизация: `auth.getTokenSecure`, `auth.getSessionSecure`, `auth.logout`, `getViewerId`.

Профили: `getProfiles`, `users.get`, `users.search`, `users.getSubscriptions`, `status.get`,
`status.set`, `getCounters`, `activity.online`, `getGroupsFull`, `groups.getById`.

Друзья: `friends.get`, `friends.getOnline`, `friends.getMutual`, `friends.getRequests`,
`friends.add`, `friends.delete`, `friends.areFriends`, `friends.getByPhones`.

Стена: `wall.get`, `wall.getById`, `wall.post`, `wall.edit`, `wall.delete`, `wall.getComments`,
`wall.addComment`, `wall.deleteComment`, `wall.addLike`, `wall.deleteLike`, `likes.add`,
`likes.delete`, `polls.getById`, `polls.addVote`.

Сообщения: `messages.getDialogs`, `messages.get`, `messages.getHistory`, `messages.getById`,
`messages.send`, `messages.delete`, `messages.markAsRead`, `messages.search`,
`messages.getLongPollServer`.

Медиа: `photos.get`, `photos.getUserPhotos`, `photos.getById`, `photos.getAlbums`,
`photos.createAlbum`, `photos.editAlbum`, `photos.deleteAlbum`, `photos.delete`, `photos.edit`,
`photos.getComments`, `photos.createComment`, `photos.getTags`, `photos.getUploadServer`,
`photos.getWallUploadServer`, `photos.save`, `photos.saveWallPhoto`, `audio.get`, `audio.getById`,
`audio.getRecommendations`, `audio.search`, `audio.getAlbums`, `audio.add`, `audio.delete`,
`audio.getUploadServer`, `audio.save`, `video.get`, `video.save`, `video.getComments`,
`notes.getById`, `notes.getComments`, `docs.getUploadServer`, `docs.save`.

Прочее: `newsfeed.get`, `newsfeed.getComments`, `places.search`, `places.getById`, `places.add`,
`places.checkin`, `places.getCheckins`, `places.getCityById`, `places.getCountryById`,
`captcha.force`, `execute`.

## Формат ответов

Ответы повторяют формат API 3.0, который ожидает клиент: массивы с первым элементом-счётчиком
(`wall.get`, `messages.getDialogs`, `friends.getRequests` без счётчика), вложения в виде
`{"type": "...", "<type>": {...}}`, `photo_rec`/`photo_medium_rec`, `performer` у аудио,
`chat_active` и `title` у бесед, обновления long poll `[4, mid, flags, peer, date, title, body]`,
`[2|3, mid, mask, peer]`, `[8|9, -uid]`.

`newsfeed.get` учитывает `end_time`/`start_time` и сортирует записи по дате — клиент листает
ленту именно этими параметрами, поэтому страницы не пересекаются и посты не дублируются.

Ссылки на загруженные файлы формируются как `/media/upload/<hash><расширение>`. Расширение
важно: клиент вычисляет имя файла в кэше как `md5(url) + расширение`, а `MediaPlayer`
определяет формат по URL с расширением и `Content-Type`. Без расширения аватарка и фото не
загружались, а музыка не воспроизводилась.

## Тесты

```
go test ./...
```

Покрыты авторизация, `execute` с проекцией `@.field`, стена, комментарии, лайки, опросы,
сообщения и long poll, беседы, загрузка файлов в стиле клиента, альбомы, медиа,
постраничная выдача ленты без дублей, а также веб-версия: регистрация, вход, загрузка
аватарки, запись с фото и с аудио-вложением, загрузка музыки, видео, заметок и документов,
альбомы, просмотр записи с лайками и комментариями, профиль, друзья и заявки, сообщения
с выбором собеседника из друзей, места с отметками, редактирование профиля, сохранение
расширения в URL медиа и поддержка `Range` для аудио. Отдельно проверяется, что абсолютные
ссылки медиа, upload-серверов и long poll строятся от хоста запроса клиента, а загруженный
трек отдаётся с `audio/mpeg`. Для анти-DDoS проверяются ограничение всплеска, раздельные
лимиты для разных IP, отклонение лишних одновременных обработок и обход лимита для long poll.
