local wheelName = KEYS[1]
local totalSegments = ARGV[1]

local spins = redis.call('INCRBY', 'wheel:' .. wheelName .. ':spins', 0)
local enabled = redis.call('INCRBY', 'wheel:' .. wheelName .. ':enabled', 0)

local prizes = {}
for id=1, totalSegments, 1 do
    prizes[id] = redis.call('INCRBY', 'wheel:' .. wheelName .. ':segment:'.. id, 0)
end

return {spins, prizes, enabled}