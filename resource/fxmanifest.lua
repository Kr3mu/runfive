fx_version "cerulean"
game "gta5"
lua54 "yes"

author "Kr3mu"
description "runfive ingame panel"
version "0.0.1"

shared_scripts {
    "@ox_lib/init.lua",
    "config/main.lua",
}

client_scripts { "client/main.lua" }
server_scripts { "server/main.lua" }

dependencies { "ox_lib" }
