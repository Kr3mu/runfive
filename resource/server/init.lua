--[[
Resource server entry point. Boots subsystems in order and holds them
registered for the lifetime of the resource.
]]

local registry = lib.require("server.adapters.init")

-- Detection reads information_schema via oxmysql, which needs the DB
-- connection to be up. MySQL.ready fires once oxmysql has confirmed the
-- connection string resolves; running detection before that yields false
-- negatives.
MySQL.ready(function()
    registry.detect()
end)
