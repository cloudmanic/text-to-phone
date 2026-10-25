<!--
=============================================================================
File: README.md
Author: Spicer Matthews <spicer@cloudmanic.com>
Created: 2026-10-24
Copyright: 2026 Cloudmanic, LLC. All rights reserved.
=============================================================================
-->

# text-to-phone

Post text to `POST /call`. Twilio calls `CALL_TO_NUMBER` from `TWILIO_FROM_NUMBER`
and reads the text out loud when the phone is answered.

## Run

```bash
cp .env.example .env   # then fill in the values
go build -o text-to-phone . && ./text-to-phone
```

## Call it

```bash
curl -X POST "http://localhost:8080/call?access_token=YOUR_TOKEN" \
  --data-urlencode "text=Hello from the app"
```

`access_token` and `text` can be in the query string, a form body, or JSON
(`{"text": "...", "access_token": "..."}`). A plain-text body is read as the text.

| Result | When |
|---|---|
| 200 | Call placed. Returns `call_sid` and `status`. |
| 400 | No `access_token`, no `text`, or text too long. |
| 401 | Wrong `access_token`. |
| 405 | Not a POST. |
| 502 | Twilio refused the call (see the server log). |
