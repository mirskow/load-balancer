local stateKey = KEYS[1]       -- bucket:<client_id>
local configKey = KEYS[2]      -- config:<client_id>
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
local defaultCapacity = tonumber(ARGV[3])
local defaultRate = tonumber(ARGV[4])

-- Прочитать конфиг, если нет — установить дефолт
local config = redis.call("HMGET", configKey, "capacity", "rate")
local capacity = tonumber(config[1])
local rate = tonumber(config[2])

if not capacity then
    capacity = defaultCapacity
    redis.call("HSET", configKey, "capacity", capacity)
end

if not rate then
    rate = defaultRate
    redis.call("HSET", configKey, "rate", rate)
end

-- Прочитать состояние
local state = redis.call("HMGET", stateKey, "tokens", "last")
local tokens = tonumber(state[1])
local last = tonumber(state[2])

-- Если клиент пришел в первый раз, то устанавливаем начальное количество токенов
if not tokens or not last then
    redis.call("HSET", stateKey, "tokens", capacity - 1, "last", now)
    redis.call("EXPIRE", stateKey, ttl)
    return 1
end

-- Проверка на недостаток токенов
if tokens < 1 then
    redis.call("HSET", stateKey, "tokens", tokens, "last", now)
    redis.call("EXPIRE", stateKey, ttl)
    return 0
else
    -- Вытянуть один токен
    tokens = tokens - 1
    redis.call("HSET", stateKey, "tokens", tokens, "last", now)
    redis.call("EXPIRE", stateKey, ttl)
    return 1
end
