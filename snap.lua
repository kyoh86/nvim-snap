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
local init = require("snap.command_init")
local list = require("snap.command_list")
local new_case = require("snap.command_new")
local run = require("snap.command_run")
local suite_compare = require("snap.command_suite_compare")
local update_expected = require("snap.command_update_expected")
local util = require("snap.util")

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua [command] [options]",
    "",
    "commands:",
    "  list             List test cases (high-level)",
    "  new              Create a new test case (high-level)",
    "  init             Scaffold CI workflow",
    "  run              Run test cases (high-level)",
    "  compare          Compare test cases (high-level)",
    "  update-expected  Update expected snapshots (high-level)",
    "  core             Low-level commands (capture/normalize/compare)",
    "",
    "run 'nvim -l snap.lua core capture --help' for low-level options",
  }, "\n")
end

function M.main(args_list)
  local args = vim.deepcopy(args_list or {})
  local command = args[1]
  if command == nil or vim.startswith(command, "-") then
    util.err_write("command is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  table.remove(args, 1)
  if command == "list" then
    return list.run(args)
  end
  if command == "new" then
    return new_case.run(args)
  end
  if command == "init" then
    return init.run(args)
  end
  if command == "run" then
    return run.run(args)
  end
  if command == "compare" then
    return suite_compare.run(args)
  end
  if command == "update-expected" then
    return update_expected.run(args)
  end
  if command == "core" then
    local core_cmd = args[1]
    if core_cmd == nil or vim.startswith(core_cmd, "-") then
      util.err_write("core command is required")
      util.err_write("usage: nvim -l snap.lua core [capture|normalize|compare] [options]")
      vim.cmd("cq")
      return
    end
    table.remove(args, 1)
    if core_cmd == "capture" then
      return capture.run(args)
    end
    if core_cmd == "normalize" then
      return normalize.run(args)
    end
    if core_cmd == "compare" then
      return compare.run(args)
    end
    util.err_write("unknown core command: " .. core_cmd)
    util.err_write("usage: nvim -l snap.lua core [capture|normalize|compare] [options]")
    vim.cmd("cq")
    return
  end
  if command == "help" or command == "--help" or command == "-h" then
    print(usage())
    return
  end
  util.err_write("unknown command: " .. command)
  util.err_write(usage())
  vim.cmd("cq")
end

local ok, err = pcall(M.main, _G.arg or {})
if not ok then
  util.err_write(err)
  vim.cmd("cq")
end
