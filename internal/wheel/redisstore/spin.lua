-- Atomically picks a winning segment, consumes its stock and records the spin.
-- The weighting rules must match Wheel.Weights in internal/wheel/wheel.go.
--
-- KEYS[1] def     JSON wheel definition
-- KEYS[2] state   hash: enabled, spins
-- KEYS[3] stock   hash: segment index -> remaining prizes (limited segments only)
-- KEYS[4] wins    hash: segment index -> wins
-- KEYS[5] history list of JSON spin records, newest first
--
-- ARGV[1] uniform random value in [0, 1)
-- ARGV[2] user uid
-- ARGV[3] user display name
-- ARGV[4] spin time, unix milliseconds
-- ARGV[5] max history length
--
-- Returns the zero based winning index, or a negative status:
-- -1 not found, -2 disabled, -3 exhausted.

local raw = redis.call('GET', KEYS[1])
if not raw then
  return -1
end
if redis.call('HGET', KEYS[2], 'enabled') ~= '1' then
  return -2
end

local def = cjson.decode(raw)
local weights = {}
local limited = {}
local total = 0

for i, seg in ipairs(def.segments) do
  local field = tostring(i - 1)
  local w = 0
  local isLimited = seg.stock ~= nil and seg.stock ~= cjson.null
  local remaining = 0
  if isLimited then
    remaining = tonumber(redis.call('HGET', KEYS[3], field) or '0')
  end
  if not isLimited or remaining > 0 then
    if def.mode == 'equal' then
      w = 1
    elseif def.mode == 'weighted' then
      w = tonumber(seg.weight) or 1
    elseif def.mode == 'inventory' then
      w = remaining
    end
  end
  weights[i] = w
  limited[i] = isLimited
  total = total + w
end

if total <= 0 then
  return -3
end

local target = tonumber(ARGV[1]) * total
local winner = 0
local acc = 0
for i, w in ipairs(weights) do
  if w > 0 then
    winner = i
    acc = acc + w
    if target < acc then
      break
    end
  end
end

local field = tostring(winner - 1)
if limited[winner] then
  redis.call('HINCRBY', KEYS[3], field, -1)
end
redis.call('HINCRBY', KEYS[4], field, 1)
redis.call('HINCRBY', KEYS[2], 'spins', 1)

redis.call('LPUSH', KEYS[5], cjson.encode({
  i = winner - 1,
  uid = ARGV[2],
  name = ARGV[3],
  t = tonumber(ARGV[4]),
}))
redis.call('LTRIM', KEYS[5], 0, tonumber(ARGV[5]) - 1)

return winner - 1
