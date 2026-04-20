--[[
LuaCATS type annotations for the framework-adapter layer. Pure documentation —
Lua has no runtime interface enforcement, so these exist for lua-language-server
diagnostics and as the canonical reference for what each adapter must expose.
]]

---@alias AdapterId
---| "esx_legacy"
---| "esx_1x"
---| "qbcore"
---| "qbox"
---| "nd_core"
---| "ox_core"
---| "vrp"
---| "unknown"

---@class AdapterCapabilities
---@field extraJobs      boolean  -- framework supports more than one job per character
---@field sharedAccounts boolean  -- framework has money accounts shared across characters
---@field multiChar      boolean  -- one license can own multiple characters
---@field softDelete     boolean  -- framework marks characters as deleted without removing rows

---@class DetectionDetails
---@field resource?  string      -- GetResourceState("<framework_resource>")
---@field markers?   string[]    -- human-readable list of SQL markers that matched
---@field note?      string      -- arbitrary extra detail for debug output

---@class MoneyBlock
---@field cash?        integer
---@field bank?        integer
---@field black_money? integer   -- ESX only
---@field crypto?      integer   -- QBCore / Qbox only

---@class JobBlock
---@field name       string
---@field label?     string
---@field grade      integer
---@field gradeName? string
---@field gradeLabel? string
---@field salary?    integer
---@field onDuty?    boolean     -- QBCore / Qbox
---@field isBoss?    boolean

---@class ExtraJob
---@field name      string
---@field label?    string
---@field grade     integer
---@field isActive  boolean

---@class AccountEntry  -- OX Core shared accounts; empty for every other framework
---@field id         integer
---@field label?     string
---@field balance    integer
---@field type       "personal" | "shared" | "group" | "inactive"
---@field isDefault  boolean
---@field role?      string

---@class InventoryItem
---@field name      string
---@field count     integer
---@field slot?     integer
---@field metadata? table

---@class PlayerName
---@field first?   string
---@field last?    string
---@field display  string

---@class Position
---@field x        number
---@field y        number
---@field z        number
---@field heading? number

---@class PlayerSummary   -- cheap shape returned by listPlayers()
---@field license       string
---@field frameworkPk   string
---@field name          PlayerName
---@field job?          { name: string, grade: integer }
---@field lastUpdated?  integer

---@class PlayerRecord
---@field license        string
---@field license2?      string
---@field steam?         string
---@field discord?       string
---@field fivem?         string
---@field framework      AdapterId
---@field frameworkPk    string
---@field characterSlot? integer
---@field name           PlayerName
---@field dob?           string
---@field gender?        string
---@field phone?         string
---@field money          MoneyBlock
---@field accounts       AccountEntry[]
---@field job?           JobBlock
---@field extraJobs      ExtraJob[]
---@field inventory      InventoryItem[]
---@field metadata       table
---@field position?      Position
---@field lastSeen?      integer
---@field lastUpdated?   integer
---@field isDeleted      boolean
---@field fetchedAt      integer   -- os.time() when the adapter returned this record

---@class Adapter
---@field id            AdapterId
---@field priority      integer   -- lower wins when multiple adapters claim
---@field capabilities  AdapterCapabilities
---@field detect        fun(): boolean, DetectionDetails?
---@field listPlayers   fun(): PlayerSummary[]
---@field getPlayer     fun(license: string): PlayerRecord?
---@field getPlayerByPk fun(pk: string): PlayerRecord?
---@field getMoney      fun(pk: string): MoneyBlock?
---@field getJob        fun(pk: string): JobBlock?
---@field getInventory  fun(pk: string): InventoryItem[]?
---@field getMetadata   fun(pk: string): table?
---@field forceSave?    fun(pk: string): boolean?   -- optional; triggers framework-side Save() via export
