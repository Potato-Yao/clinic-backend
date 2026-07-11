What clinic_proxy_dingtalk sends to the backend
Target URL construction (Program.cs:135-136)
{backend config value} + {url header value} + ?username={cached job_number}
- backend comes from appsettings.json backend key — e.g. https://clinic.bitnp.net
- url is a header the mini-program sends, e.g. /api/wechat/ or /api/wechat/working/
- ?username= is always appended; the mini-program never controls it (uses the cached job_number from DingTalk login)
  Examples of final URLs the backend sees:
  Mini-program sends url: header	Backend receives
  /api/wechat/	{backend}/api/wechat/?username=1120221234
  /api/wechat/working/	{backend}/api/wechat/working/?username=1120221234
  /api/wechat/finish/	{backend}/api/wechat/finish/?username=1120221234
  /api/wechat/15/	{backend}/api/wechat/15/?username=1120221234
  Headers on the forwarded request (Program.cs:139-144)
  Always — every method:
  Header	Value	How computed
  X-API-KEY	<32-char lowercase hex md5>	md5(_apikey + cachedUserName + timeStr) where _apikey is shared secret (matches Django settings.py:227 apikey) and timeStr is the same string used for Date below
  Date	RFC1123 string e.g. Thu, 09 Jul 2026 10:00:00 GMT	DateTime.Now.ToString("R") — this exact value is also the 3rd md5 input
  X-Forwarded-For	client IP	context.Connection.RemoteIpAddress (the student's real IP, not proxy)
  Accept	whatever mini-program sent	passthrough
  For POST/PUT only:
  Header	Value
  Content-Type	whatever mini-program sent (usually application/json)
  Body transformation (POST/PUT only) — Program.cs:355-374
  parse incoming JSON body
  if no "user" key:
  add "user" = <cached job_number>
  re-serialize and send
  So for a booking, backend receives JSON body like:
  {
  "realname":         "张三",
  "phone_num":        "13800138000",
  "campus":           "中关村",
  "appointment_time": "2026-07-15",
  "description":      "笔电不开机",
  "model":            "ThinkPad X1",
  "password":         "device-pw-or-empty",
  "user":             "1120221234"
  }
  The user field may or may not be present (mini-program omits it; the proxy adds it). If present, the proxy keeps the original value, but backend trust should be on ?username= since that's what's used for the md5 signature.
  GET and DELETE send no body.
  Per-method summary
  Method	URL pattern	Has body?	Handler behavior
  GET	{backend}{url}?username={jn}	No	read-only fetch
  POST	{backend}{url}?username={jn}	Yes — JSON, user injected if missing	creation (booking, etc.)
  PUT	{backend}{url}?username={jn}	Yes — same	update (rarely used)
  DELETE	{backend}{url}?username={jn}	No	cancellation
  Concrete booking example — complete wire format
  POST https://clinic.bitnp.net/api/wechat/?username=1120221234 HTTP/1.1
  X-API-KEY: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
  Date: Thu, 09 Jul 2026 10:00:00 GMT
  X-Forwarded-For: 10.3.2.1
  Accept: application/json, text/plain, */*
  Content-Type: application/json

{
"realname": "张三",
"phone_num": "13800138000",
"campus": "中关村",
"appointment_time": "2026-07-15",
"description": "笔电不开机",
"model": "ThinkPad X1",
"password": "",
"user": "1120221234"
}
What your Go middleware must verify
dateHeader := c.GetHeader("Date")                  // RFC1123
username   := c.Query("username")                  // job_number
clientKey  := c.GetHeader("X-API-KEY")             // 32-hex

expected := md5(SHARED_SECRET + username + dateHeader)
if !hmac.Equal([]byte(expected), []byte(clientKey)) → 401

// optional: reject if dateHeader is stale (>5min from now) to prevent replay
t, err := time.Parse(time.RFC1123, dateHeader)
if err != nil || time.Since(t) > 5*time.Minute → 401

// find or create User{username: username}, set as c.MustGet("user")
Then routes:
- POST /api/tickets → create record (run the 4-step validation from views.py:88)
- GET /api/tickets/working → return user's in-progress record or {"count":0}
- GET /api/tickets/finished → list finished records for the user
- GET /api/tickets/:id → fetch one
- DELETE /api/tickets/:id → cancel with date-still-open check
  And the legacy alias POST /api/wechat, GET /api/wechat/working, etc. pointing at the same handlers (per the previous discussion).