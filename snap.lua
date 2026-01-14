local M = {}

local function add_package_path()
  local source = debug.getinfo(1, "S").source
  if type(source) ~= "string" then
    return
  end
  if vim.startswith(source, "@") then
    source = source:sub(2)
  end
  local dir = vim.fs.dirname(source)
  if not dir or dir == "" then
    return
  end
  local extra = table.concat({
    dir .. "/?.lua",
    dir .. "/?/init.lua",
  }, ";")
  package.path = extra .. ";" .. package.path
end

add_package_path()

local capture = require("snap.command_capture")
local normalize = require("snap.command_normalize")
local compare = require("snap.command_compare")
local util = require("snap.util")

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua [command] [options]",
    "",
    "commands:",
    "  capture   Capture a UI snapshot (default)",
    "  normalize Normalize snapshot JSON",
    "  compare   Compare snapshot JSON",
    "",
    "run 'nvim -l snap.lua capture --help' for capture options",
  }, "\n")
end

function M.main(args_list)
  local args = vim.deepcopy(args_list or {})
  local command = args[1]
  if command == nil or vim.startswith(command, "-") then
    return capture.run(args)
  end
  table.remove(args, 1)
  if command == "capture" then
    return capture.run(args)
  end
  if command == "normalize" then
    return normalize.run(args)
  end
  if command == "compare" then
    return compare.run(args)
  end
  if command == "help" or command == "--help" or command == "-h" then
    print(usage())
    return
  end
  util.err_write("unknown command: " .. command)
  util.err_write(usage())
  vim.cmd("cq")
end

if ... then
  return M
end

local ok, err = pcall(M.main, _G.arg or {})
if not ok then
  util.err_write(err)
  vim.cmd("cq")
end
