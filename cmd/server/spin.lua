local wheelName = KEYS[1]
local randomVal = ARGV[1]
local totalSegments = ARGV[2]

local prizeTotal = 0
local segments = {}
local scaledSegments = {}
local prevRange = 0

for i=1, totalSegments, 1  do
    local v = redis.call('incrby','wheel:' .. wheelName .. ':segment:' .. i, 0)
    prizeTotal = prizeTotal + v
    segments[i] = v

    scaledSegments[i] = prevRange + v
    prevRange = scaledSegments[i]
end

if prizeTotal == 0 then
    return {0, 0}
end

if prizeTotal == 1 then
    redis.call('decrby', 'wheel:' .. wheelName .. ':enabled', 1)
end

local rand = randomVal * prizeTotal

local winningSegmentIndex = 0
for i=1, totalSegments, 1  do
    if rand < scaledSegments[i] then
        winningSegmentIndex = i
        break
    end
end

if segments[winningSegmentIndex] <= 0 then
    return {winningSegmentIndex, prizeTotal, "outOfBounds"}
end

redis.call('decr', 'wheel:' .. wheelName .. ':segment:' .. winningSegmentIndex)
redis.call('incrby','wheel:' .. wheelName .. ':spins', 1)
return {winningSegmentIndex, prizeTotal}