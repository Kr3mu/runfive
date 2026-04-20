--[[
Adapter registry. Loads known adapter modules, runs priority-ordered detection
once on startup, and dispatches adapter-method calls through a pcall wrapper
with structured logging.

Framework adapters (qbcore, esx, nd_core, ox_core, vrp) will be added in their
own tickets — see #34-#38. For now only the `unknown` fallback is registered.
]]

local helpers = lib.require("server.adapters._helpers")

local M = {}

---@type Adapter[]
local adapters = {}

---@type Adapter?
local active = nil

---@type DetectionDetails?
local activeDetails = nil

-----------------------------------------------------------------------------
-- Registration. Adapters are added in the order they should be TRIED at
-- detection time; `priority` on the module is the tiebreaker.
-----------------------------------------------------------------------------

---@param adapter Adapter
function M.register(adapter)
    assert(adapter and adapter.id, "adapter module must expose .id")
    table.insert(adapters, adapter)
end

-----------------------------------------------------------------------------
-- Structured logging. One JSON line per event for easy grep / ingest.
-- Logs go to the server console via FiveM's print; ingestion layer (panel or
-- external log shipper) can parse the `[adapter] {...}` prefix if needed.
-----------------------------------------------------------------------------

---@param level "info" | "warn" | "error"
---@param event string
---@param fields table?
local function log(level, event, fields)
    fields = fields or {}
    fields.level = level
    fields.event = event
    fields.ts    = os.time()
    print("[adapter] " .. json.encode(fields))
end

-----------------------------------------------------------------------------
-- Detection pipeline. Sorts registered adapters by priority, asks each
-- `detect()` in turn, first match wins. The `unknown` adapter always returns
-- true at priority 999 so we always end up with an active adapter.
-----------------------------------------------------------------------------

---@return Adapter, DetectionDetails?
local function runDetection()
    table.sort(adapters, function(a, b) return a.priority < b.priority end)

    for _, adapter in ipairs(adapters) do
        local ok, matched, details = pcall(adapter.detect)
        if not ok then
            log("error", "detect_threw", {
                adapter_id = adapter.id,
                error      = tostring(matched),
            })
        elseif matched then
            log("info", "detect_matched", {
                adapter_id = adapter.id,
                details    = details,
            })
            return adapter, details
        end
    end

    error("no adapter claimed the server — unknown fallback is missing from the registry")
end

---Runs detection. Safe to call from startup and from a rescan trigger.
function M.detect()
    helpers.invalidateSchemaCache()
    active, activeDetails = runDetection()
    return active, activeDetails
end

-----------------------------------------------------------------------------
-- Accessors.
-----------------------------------------------------------------------------

---@return Adapter?
function M.current() return active end

---@return DetectionDetails?
function M.currentDetails() return activeDetails end

-----------------------------------------------------------------------------
-- Dispatcher. Wraps every public adapter call in pcall so adapter bugs don't
-- take down the resource. Failures emit a structured log and return nil, which
-- the caller (event emitters, internal API handlers) treats as
-- "framework data unavailable".
-----------------------------------------------------------------------------

---@param method string   -- one of "listPlayers", "getPlayer", "getMoney", ...
---@vararg any
---@return any
function M.invoke(method, ...)
    if not active then
        log("warn", "invoke_before_detect", { method = method })
        return nil
    end
    local fn = active[method]
    if type(fn) ~= "function" then
        return nil
    end
    local ok, result = pcall(fn, ...)
    if not ok then
        log("error", "invoke_threw", {
            adapter_id = active.id,
            method     = method,
            error      = tostring(result),
        })
        return nil
    end
    return result
end

-----------------------------------------------------------------------------
-- Built-in adapters. Framework-specific modules are registered here as they
-- land in #34-#38.
-----------------------------------------------------------------------------

M.register(lib.require("server.adapters.unknown"))

return M
