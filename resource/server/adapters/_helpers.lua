--[[
Shared helpers used by every framework adapter. See _contract.lua for types.
]]

local M = {}

-----------------------------------------------------------------------------
-- JSON: safeDecode never throws, returns nil + logs on malformed input.
-----------------------------------------------------------------------------

---@param str string|nil
---@return any decoded, string? err
function M.safeDecode(str)
    if str == nil or str == "" then return nil end
    if type(str) ~= "string" then return str end
    local ok, result = pcall(json.decode, str)
    if not ok then
        return nil, tostring(result)
    end
    return result
end

-----------------------------------------------------------------------------
-- Schema probes: tableExists / columnExists / columnType, cached 60 seconds.
-- The cache key survives DROP/CREATE within the window, but that's acceptable
-- for our use case — schema changes mid-runtime are not normal operation, and
-- a 60s stale read is still much fresher than a missing probe.
-----------------------------------------------------------------------------

local TTL = 60
local cache = {}

---@param key string
---@param fetch fun(): any
---@return any
local function cached(key, fetch)
    local entry = cache[key]
    local now = os.time()
    if entry and entry.expiresAt > now then
        return entry.value
    end
    local value = fetch()
    cache[key] = { value = value, expiresAt = now + TTL }
    return value
end

---@param tableName string
---@return boolean
function M.tableExists(tableName)
    return cached("t:" .. tableName, function()
        local row = MySQL.scalar.await(
            [[SELECT 1 FROM information_schema.TABLES
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? LIMIT 1]],
            { tableName }
        )
        return row == 1
    end)
end

---@param tableName string
---@param columnName string
---@return boolean
function M.columnExists(tableName, columnName)
    return cached("c:" .. tableName .. ":" .. columnName, function()
        local row = MySQL.scalar.await(
            [[SELECT 1 FROM information_schema.COLUMNS
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1]],
            { tableName, columnName }
        )
        return row == 1
    end)
end

---@param tableName string
---@param columnName string
---@return string? dataType  -- nil if column does not exist; "int", "longtext", "varchar", ...
function M.columnType(tableName, columnName)
    return cached("ct:" .. tableName .. ":" .. columnName, function()
        return MySQL.scalar.await(
            [[SELECT DATA_TYPE FROM information_schema.COLUMNS
              WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1]],
            { tableName, columnName }
        )
    end)
end

---Forget cached schema probes. Called by the registry on adapter rescan.
function M.invalidateSchemaCache()
    cache = {}
end

-----------------------------------------------------------------------------
-- License normalisation. Canonical form is "license:<hex>" with prefix.
-----------------------------------------------------------------------------

---@param id string|nil
---@return string? hex
function M.stripLicensePrefix(id)
    if not id then return nil end
    local hex = id:match("^license:(.+)$")
    return hex or id
end

---@param id string|nil
---@return string? canonical
function M.normalizeLicense(id)
    if not id or id == "" then return nil end
    if id:find("^license:") then return id end
    return "license:" .. id
end

-----------------------------------------------------------------------------
-- ox_inventory reader. Used by every framework that defaults to ox_inventory
-- for inventory storage. Owner is the framework's own PK (identifier for ESX,
-- citizenid for QB/Qbox, charid for ND, charId for OX Core).
-----------------------------------------------------------------------------

---@param owner string|integer
---@return InventoryItem[]
function M.readOxInventory(owner)
    local row = MySQL.single.await(
        "SELECT data FROM ox_inventory WHERE owner = ? LIMIT 1",
        { tostring(owner) }
    )
    if not row or not row.data then return {} end
    local decoded = M.safeDecode(row.data)
    if type(decoded) ~= "table" then return {} end
    return decoded
end

---@param owner string|integer
---@return boolean
function M.oxInventoryActive()
    return GetResourceState("ox_inventory") == "started"
end

return M
