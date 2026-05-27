-- Global request configuration
wrk.method = "GET"
wrk.path   = "/api/wallet/balance"

-- Pick ONE authentication method — Authorization header is preferred
wrk.headers = {
   ["Accept"]          = "*/*",
   ["Accept-Language"] = "en,vi;q=0.9",
   ["Authorization"]   = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjkxNTQwNzcsImlhdCI6MTc2OTA2NzY3NywidXNlcm5hbWUiOiJraGFuZ2RjIn0.J6rSkr3H8X2f72k-WxxkO5hYHltCjb3EGT2nLRWTaP4",
   ["Connection"]      = "keep-alive",
   ["token"]           = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjkxNTQwNzcsImlhdCI6MTc2OTA2NzY3NywidXNlcm5hbWUiOiJraGFuZ2RjIn0.J6rSkr3H8X2f72k-WxxkO5hYHltCjb3EGT2nLRWTaP4",
   ["DNT"]             = "1",
   ["Referer"]         = "http://localhost:8081/dashboard",
   ["User-Agent"]      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
}

-- Optional: very clean minimal version (usually enough)
-- wrk.headers = {
--    ["Authorization"] = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
--    ["Accept"]        = "application/json",
-- }

-- Debug helper – prints only failed requests
response = function(status, headers, body)
   if status < 200 or status > 299 then
      print("┌─ Failed request")
      print("│  Status: " .. status)
      print("│  Body:   " .. (body or "<empty>"))
      print("└──────────────")
   end
end