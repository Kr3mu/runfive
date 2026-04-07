fx_version "cerulean"
game "gta5"
lua54 "yes"

author "Kr3mu"
description "runfive ingame panel"
version "0.0.1"

shared_scripts {
    "@ox_lib/init.lua",
    "config/shared.lua",
}

client_scripts { "client/init.lua" }
server_scripts { "server/init.lua" }

dependencies { "ox_lib" }
