local stateKey = KEYS[1]       -- bucket:<client_id>
local configKey = KEYS[2]      -- config:<client_id>
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

-- Получаем конфигурацию
local config = redis.call("HMGET", configKey, "capacity", "rate")
if not config or #config ~= 2 then
    return redis.error_reply("Invalid config data")
end

local capacity = tonumber(config[1])
local rate = tonumber(config[2])

-- Проверяем, что capacity и rate являются числами
if not capacity or not rate then
    return redis.error_reply("Invalid capacity or rate values")
end

-- Получаем состояние
local state = redis.call("HMGET", stateKey, "tokens", "last")
if not state or #state ~= 2 then
    return redis.error_reply("Invalid state data")
end

local tokens = tonumber(state[1])
local last = tonumber(state[2])

-- Проверяем, что tokens и last являются числами
if not tokens or not last then
    return redis.error_reply("Invalid tokens or last values")
end

-- Рассчитываем количество прошедших секунд
local elapsed = now - last
tokens = math.min(capacity, tokens + elapsed * rate)

-- Обновляем состояние в Redis
redis.call("HMSET", stateKey, "tokens", tokens, "last", now)
redis.call("EXPIRE", stateKey, ttl)

return tokens

