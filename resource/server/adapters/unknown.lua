--[[
Fallback adapter. Always detects last (priority 999) when every other adapter
has declined. Returns session-only records — no framework data, just the
identity we already know from GetPlayerIdentifiers().
]]

---@type Adapter
local M = {
    id = "unknown",
    priority = 999,
    capabilities = {
        extraJobs      = false,
        sharedAccounts = false,
        multiChar      = false,
        softDelete     = false,
    },
}

function M.detect()
    return true, { note = "no framework markers matched; session-only data" }
end

function M.listPlayers()
    return {}
end

---@param license string
function M.getPlayer(license)
    return {
        license     = license,
        framework   = "unknown",
        frameworkPk = license,
        name        = { display = "" },
        money       = {},
        accounts    = {},
        extraJobs   = {},
        inventory   = {},
        metadata    = {},
        isDeleted   = false,
        fetchedAt   = os.time(),
    }
end

function M.getPlayerByPk(pk)
    return M.getPlayer(pk)
end

function M.getMoney(_)     return nil end
function M.getJob(_)       return nil end
function M.getInventory(_) return {}  end
function M.getMetadata(_)  return {}  end

return M
