-- Bundled by luabundle {"version":"1.7.0"}
local __bundle_require, __bundle_loaded, __bundle_register, __bundle_modules = (function(superRequire)
	local loadingPlaceholder = {[{}] = true}

	local register
	local modules = {}

	local require
	local loaded = {}

	register = function(name, body)
		if not modules[name] then
			modules[name] = body
		end
	end

	require = function(name)
		local loadedModule = loaded[name]

		if loadedModule then
			if loadedModule == loadingPlaceholder then
				return nil
			end
		else
			if not modules[name] then
				if not superRequire then
					local identifier = type(name) == 'string' and '\"' .. name .. '\"' or tostring(name)
					error('Tried to require ' .. identifier .. ', but no such module has been registered')
				else
					return superRequire(name)
				end
			end

			loaded[name] = loadingPlaceholder
			loadedModule = modules[name](require, loaded, register, modules)
			loaded[name] = loadedModule
		end

		return loadedModule
	end

	return require, loaded, register, modules
end)(require)
__bundle_register("__root", function(require, _LOADED, __bundle_register, __bundle_modules)
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
local ci = require("snap.command_ci")
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
    "  ci               CI helpers (init)",
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
  if command == "ci" then
    return ci.run(args)
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

if ... then
  return M
end

local ok, err = pcall(M.main, _G.arg or {})
if not ok then
  util.err_write(err)
  vim.cmd("cq")
end

end)
__bundle_register("snap.util", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

function M.rpc_error_to_string(err)
  if err == nil or err == vim.NIL then
    return "unknown error"
  end
  if type(err) == "table" then
    return vim.inspect(err)
  end
  return tostring(err)
end

function M.err_write(msg)
  if msg == nil then
    msg = "unknown error"
  end
  io.stderr:write(tostring(msg), "\n")
end

function M.normalize_path(base, path)
  if not path then
    return nil
  end
  if vim.fn.fnamemodify(path, ":p") == path then
    return vim.fs.normalize(path)
  end
  return vim.fs.normalize(vim.fs.joinpath(base, path))
end

return M

end)
__bundle_register("snap.command_update_expected", function(require, _LOADED, __bundle_register, __bundle_modules)
local case_def = require("snap.case_def")
local output = require("snap.output")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua update-expected [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case ID         Filter by case id (repeatable, comma-separated)",
    "  --dry-run         Show updates without writing",
    "  --no-confirm      Skip confirmation prompt",
    "  --yes             Alias of --no-confirm",
    "  -h, --help        Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_args(args)
  local opts = {
    root = ".",
    tags = {},
    cases = {},
    dry_run = false,
    confirm = true,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--dry-run" then
      opts.dry_run = true
    elseif arg == "--no-confirm" or arg == "--yes" then
      opts.confirm = false
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function read_input(path)
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

local function build_action(c)
  if c.kind == "regression" then
    if vim.fn.filereadable(c.actual_path) ~= 1 then
      return nil, "actual not found: " .. c.actual_path
    end
    return {
      case = c,
      kind = "copy",
      src = c.actual_path,
      dst = c.expected_path,
    }
  end
  if vim.fn.filereadable(c.golden_scenario_path) ~= 1 then
    return nil, "golden scenario not found: " .. c.golden_scenario_path
  end
  return {
    case = c,
    kind = "capture",
    scenario = c.golden_scenario_path,
    dst = c.expected_path,
  }
end

local function collect_snapshot(c, scenario)
  local opts = {
    scripts = { scenario },
    width = 80,
    height = 24,
    wait = 200,
    rpc_timeout = 2000,
    nvim = "nvim",
    data_home = util.normalize_path(c.dir, ".nvim-data"),
    config_home = util.normalize_path(c.dir, ".nvim-config"),
    multigrid = false,
  }
  return snapshot.collect(opts)
end

local function describe_action(action)
  if action.kind == "copy" then
    return action.src .. " -> " .. action.dst
  end
  return action.scenario .. " -> " .. action.dst
end

local function print_actions(actions)
  for _, action in ipairs(actions) do
    print(action.case.id .. "\t" .. describe_action(action))
  end
end

local function confirm_actions(actions)
  local message = string.format("Update expected snapshots for %d case(s)?", #actions)
  local choice = vim.fn.confirm(message, "&Yes\n&No", 2)
  return choice == 1
end

local function perform_action(action)
  local c = action.case
  local ok, err = case_def.ensure_dir(c.expected_dir)
  if not ok then
    return nil, err
  end
  if action.kind == "copy" then
    local text, read_err = read_input(action.src)
    if not text then
      return nil, read_err or "failed to read actual"
    end
    local ok_write, write_err = output.write(action.dst, text)
    if not ok_write then
      return nil, write_err or "failed to write expected"
    end
    return true
  end
  local snap, err_snap = collect_snapshot(c, action.scenario)
  if not snap then
    return nil, err_snap or "capture failed"
  end
  local encoded = vim.json.encode(snap)
  local ok_write, write_err = output.write(action.dst, encoded)
  if not ok_write then
    return nil, write_err or "failed to write expected"
  end
  return true
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases, errors = case_def.find_cases(root)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })
  local actions = {}
  local failed = #errors > 0

  for _, err in ipairs(errors) do
    util.err_write(err)
  end

  for _, c in ipairs(filtered) do
    local action, err = build_action(c)
    if not action then
      util.err_write(c.id .. ": " .. (err or "failed to build action"))
      failed = true
    else
      table.insert(actions, action)
    end
  end

  if #actions == 0 then
    if failed then
      vim.cmd("cquit 1")
    end
    return
  end

  if opts.dry_run then
    print_actions(actions)
    if failed then
      vim.cmd("cquit 1")
    end
    return
  end

  if opts.confirm and not confirm_actions(actions) then
    util.err_write("aborted")
    vim.cmd("cquit 1")
    return
  end

  for _, action in ipairs(actions) do
    local ok, err = perform_action(action)
    if not ok then
      util.err_write(action.case.id .. ": " .. (err or "failed to update expected"))
      failed = true
    else
      print(action.case.id .. "\tupdated")
    end
  end

  if failed then
    vim.cmd("cquit 1")
  end
end

return M

end)
__bundle_register("snap.snapshot", function(require, _LOADED, __bundle_register, __bundle_modules)
local grid = require("snap.grid")
local rpc = require("snap.rpc")
local util = require("snap.util")

local M = {}

function M.collect(opts)
  local state = {
    grids = {},
    hl_attrs = {},
    hl_groups = {},
    default_colors = {},
    got_flush = false,
  }

  local proc, err = rpc.start_embedded_nvim(opts)
  if not proc then
    return nil, err
  end

  local client = rpc.new_rpc_client(proc, opts)
  client.on_notification = function(method, params)
    if method ~= "redraw" then
      return
    end
    local function handle_event(name, args)
      if name == "grid_resize" then
        local grid_id, width, height = args[1], args[2], args[3]
        state.grids[grid_id] = state.grids[grid_id] or {}
        if type(width) == "number" and type(height) == "number" then
          grid.ensure_grid(state.grids[grid_id], height, width)
        end
      elseif name == "grid_clear" then
        local grid_id = args[1]
        state.grids[grid_id] = state.grids[grid_id] or {}
        grid.clear_grid(state.grids[grid_id])
      elseif name == "grid_destroy" then
        local grid_id = args[1]
        state.grids[grid_id] = nil
      elseif name == "grid_line" then
        local grid_id, row, col_start, cells = args[1], args[2], args[3], args[4]
        local g = state.grids[grid_id]
        if g then
          local r = row + 1
          local c = col_start + 1
          local current_hl = 0
          local row_cells = g.cells[r]
          if row_cells then
            for _, cell in ipairs(cells) do
              local text = cell[1]
              if cell[2] ~= nil then
                current_hl = cell[2]
              end
              local repeat_count = cell[3] or 1
              for _ = 1, repeat_count do
                row_cells[c] = { text = text, hl_id = current_hl }
                c = c + 1
              end
            end
          end
        end
      elseif name == "grid_scroll" then
        local grid_id, top, bot, left, right, rows = args[1], args[2], args[3], args[4], args[5], args[6]
        local g = state.grids[grid_id]
        if g then
          grid.scroll_grid(g, top, bot, left, right, rows)
        end
      elseif name == "default_colors_set" then
        state.default_colors = {
          rgb_fg = args[1],
          rgb_bg = args[2],
          rgb_sp = args[3],
          cterm_fg = args[4],
          cterm_bg = args[5],
        }
      elseif name == "hl_attr_define" then
        local id, rgb_attr, cterm_attr, info = args[1], args[2], args[3], args[4]
        state.hl_attrs[id] = {
          rgb_attr = rgb_attr,
          cterm_attr = cterm_attr,
          info = info,
        }
      elseif name == "hl_group_set" then
        local group, hl_id = args[1], args[2]
        state.hl_groups[group] = hl_id
      elseif name == "flush" then
        state.got_flush = true
      end
    end

    for _, ev in ipairs(params) do
      local name = ev[1]
      for i = 2, #ev do
        local args = ev[i]
        if type(args) == "table" then
          handle_event(name, args)
        end
      end
    end
  end

  local attach_opts = {
    ext_linegrid = true,
    ext_hlstate = true,
    ext_multigrid = opts.multigrid or false,
  }
  local _, attach_err = client:request("nvim_ui_attach", { opts.width, opts.height, attach_opts })
  if attach_err then
    return nil, util.rpc_error_to_string(attach_err)
  end

  for _, path in ipairs(opts.scripts) do
    local _, script_err = client:request("nvim_exec_lua", {
      "local p = ...; dofile(p)",
      { path },
    })
    if script_err then
      return nil, string.format("failed to run --script %q: %s", path, util.rpc_error_to_string(script_err))
    end
  end

  state.got_flush = false
  client:request("nvim_command", { "redraw" })
  vim.wait(opts.wait, function()
    return state.got_flush or proc.state.exited
  end, 5)

  client:notify("nvim_command", { "qa!" })
  vim.wait(opts.rpc_timeout, function()
    return proc.state.exited
  end, 10)

  local grids = {}
  for id, current_grid in pairs(state.grids) do
    table.insert(grids, {
      id = id,
      rows = current_grid.rows,
      cols = current_grid.cols,
      cells = current_grid.cells,
    })
  end
  table.sort(grids, function(a, b)
    return a.id < b.id
  end)

  local hl_attrs = {}
  for id, attr in pairs(state.hl_attrs) do
    table.insert(hl_attrs, {
      id = id,
      rgb_attr = attr.rgb_attr,
      cterm_attr = attr.cterm_attr,
      info = attr.info,
    })
  end
  table.sort(hl_attrs, function(a, b)
    return a.id < b.id
  end)

  local hl_groups = {}
  for name, hl_id in pairs(state.hl_groups) do
    table.insert(hl_groups, {
      name = name,
      hl_id = hl_id,
    })
  end
  table.sort(hl_groups, function(a, b)
    return a.name < b.name
  end)

  return {
    size = { columns = opts.width, lines = opts.height },
    default_colors = state.default_colors,
    hl_attrs = hl_attrs,
    hl_groups = hl_groups,
    grids = grids,
  }
end

return M

end)
__bundle_register("snap.rpc", function(require, _LOADED, __bundle_register, __bundle_modules)
local uv = vim.loop

local M = {}

function M.start_embedded_nvim(opts)
  local env = {}
  for key, value in pairs(uv.os_environ()) do
    if key ~= "XDG_DATA_HOME" and key ~= "XDG_CONFIG_HOME" then
      env[#env + 1] = key .. "=" .. value
    end
  end
  if opts.data_home then
    env[#env + 1] = "XDG_DATA_HOME=" .. opts.data_home
  end
  if opts.config_home then
    env[#env + 1] = "XDG_CONFIG_HOME=" .. opts.config_home
  end

  local stdin = uv.new_pipe(false)
  local stdout = uv.new_pipe(false)
  local stderr = uv.new_pipe(false)
  local state = { exited = false, exit_code = nil, exit_signal = nil }
  local args = { "--embed", "--headless", "-u", "NONE", "-i", "NONE", "-n" }
  local handle, pid = uv.spawn(opts.nvim, {
    args = args,
    env = env,
    stdio = { stdin, stdout, stderr },
  }, function(code, signal)
    state.exited = true
    state.exit_code = code
    state.exit_signal = signal
  end)
  if not handle then
    return nil, string.format("failed to spawn %s", opts.nvim)
  end
  return {
    handle = handle,
    pid = pid,
    stdin = stdin,
    stdout = stdout,
    stderr = stderr,
    state = state,
  }
end

function M.new_rpc_client(proc, opts)
  local client = {
    proc = proc,
    msgid = 0,
    buffer = "",
    unpacker = vim.mpack.Unpacker(),
    responses = {},
    rpc_timeout = opts.rpc_timeout,
    on_notification = function(_, _) end,
    stderr_chunks = {},
  }

  local function handle_message(msg)
    local kind = msg[1]
    if kind == 1 then
      local id = msg[2]
      client.responses[id] = { err = msg[3], result = msg[4] }
    elseif kind == 2 then
      local method = msg[2]
      local params = msg[3] or {}
      client.on_notification(method, params)
    elseif kind == 0 then
      local id = msg[2]
      client:send({ 1, id, "request not supported", vim.NIL })
    end
  end

  local function feed(chunk)
    if not chunk then
      return
    end
    client.buffer = client.buffer .. chunk
    local pos = 1
    while pos <= #client.buffer do
      local ok, msg, next_pos = pcall(client.unpacker, client.buffer, pos)
      if not ok then
        break
      end
      pos = next_pos
      handle_message(msg)
    end
    if pos > 1 then
      client.buffer = string.sub(client.buffer, pos)
    end
  end

  proc.stdout:read_start(function(err, chunk)
    if err then
      client.last_error = err
      return
    end
    feed(chunk)
  end)

  proc.stderr:read_start(function(_, chunk)
    if chunk then
      table.insert(client.stderr_chunks, chunk)
    end
  end)

  function client:send(msg)
    local ok, err = pcall(function()
      self.proc.stdin:write(vim.mpack.encode(msg))
    end)
    if not ok then
      return false, err
    end
    return true
  end

  function client:request(method, params)
    self.msgid = self.msgid + 1
    local id = self.msgid
    local ok, err = self:send({ 0, id, method, params or {} })
    if not ok then
      return nil, err
    end
    local done = vim.wait(self.rpc_timeout, function()
      return self.responses[id] ~= nil or self.proc.state.exited
    end, 5)
    if not done then
      return nil, "rpc timeout"
    end
    if self.proc.state.exited then
      return nil, "nvim exited"
    end
    local resp = self.responses[id]
    self.responses[id] = nil
    if resp.err ~= nil and resp.err ~= vim.NIL then
      return nil, resp.err
    end
    return resp.result
  end

  function client:notify(method, params)
    return self:send({ 2, method, params or {} })
  end

  return client
end

return M

end)
__bundle_register("snap.grid", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

function M.alloc_row(cols)
  local row = {}
  for c = 1, cols do
    row[c] = { text = " ", hl_id = 0 }
  end
  return row
end

function M.ensure_grid(grid, rows, cols)
  grid.rows = rows
  grid.cols = cols
  grid.cells = grid.cells or {}
  for r = 1, rows do
    local row = grid.cells[r]
    if not row then
      grid.cells[r] = M.alloc_row(cols)
    else
      if #row < cols then
        for c = #row + 1, cols do
          row[c] = { text = " ", hl_id = 0 }
        end
      elseif #row > cols then
        for c = cols + 1, #row do
          row[c] = nil
        end
      end
    end
  end
  for r = rows + 1, #grid.cells do
    grid.cells[r] = nil
  end
end

function M.clear_grid(grid)
  if not grid.rows or not grid.cols then
    return
  end
  for r = 1, grid.rows do
    local row = grid.cells[r]
    if not row then
      row = M.alloc_row(grid.cols)
      grid.cells[r] = row
    else
      for c = 1, grid.cols do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  end
end

function M.copy_cell(cell)
  return { text = cell.text, hl_id = cell.hl_id }
end

function M.scroll_grid(grid, top, bot, left, right, rows)
  if rows == 0 or not grid.cells then
    return
  end
  local top_r = top + 1
  local bot_r = bot
  local left_c = left + 1
  local right_c = right
  if rows > 0 then
    for r = top_r, bot_r - rows do
      local src = grid.cells[r + rows]
      local dst = grid.cells[r]
      for c = left_c, right_c do
        dst[c] = M.copy_cell(src[c])
      end
    end
    for r = bot_r - rows + 1, bot_r do
      local row = grid.cells[r]
      for c = left_c, right_c do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  else
    local offset = -rows
    for r = bot_r, top_r + offset, -1 do
      local src = grid.cells[r - offset]
      local dst = grid.cells[r]
      for c = left_c, right_c do
        dst[c] = M.copy_cell(src[c])
      end
    end
    for r = top_r, top_r + offset - 1 do
      local row = grid.cells[r]
      for c = left_c, right_c do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  end
end

return M

end)
__bundle_register("snap.output", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

function M.write(out, contents)
  if out == "-" then
    io.write(contents)
    io.write("\n")
    return true
  end
  local dir = vim.fn.fnamemodify(out, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(out, "w")
  if not fd then
    return false, err
  end
  fd:write(contents)
  fd:close()
  return true
end

return M

end)
__bundle_register("snap.case_def", function(require, _LOADED, __bundle_register, __bundle_modules)
local util = require("snap.util")

local M = {}

local function read_json(path)
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    return nil, "failed to parse case.json: " .. tostring(decoded)
  end
  if type(decoded) ~= "table" then
    return nil, "invalid case.json: expected object"
  end
  return decoded
end

local function basename(path)
  return vim.fs.basename(vim.fs.normalize(path))
end

local function normalize_tags(tags)
  if type(tags) ~= "table" then
    return {}
  end
  local out = {}
  for _, tag in ipairs(tags) do
    if type(tag) == "string" and tag ~= "" then
      table.insert(out, tag)
    end
  end
  return out
end

function M.load_case(case_path)
  local case_dir = vim.fs.normalize(vim.fn.fnamemodify(case_path, ":p:h"))
  local config, err = read_json(case_path)
  if not config then
    return nil, err
  end
  local version = config.version
  if type(version) ~= "number" or version < 1 or math.floor(version) ~= version then
    return nil, "case version is required"
  end
  local id = config.id
  if type(id) ~= "string" or id == "" then
    return nil, "case id is required"
  end
  local kind = config.kind
  if kind ~= "regression" and kind ~= "golden" then
    return nil, "case kind must be regression or golden"
  end
  local name = config.name
  if type(name) ~= "string" or name == "" then
    name = basename(case_dir)
  end
  local tags = normalize_tags(config.tags)

  local expected_dir = util.normalize_path(case_dir, "expected")
  local actual_dir = util.normalize_path(case_dir, "actual")
  local diff_dir = util.normalize_path(case_dir, "diff")

  local scenario = util.normalize_path(case_dir, "scenario.lua")
  local golden = util.normalize_path(case_dir, "golden.lua")
  local target = util.normalize_path(case_dir, "target.lua")

  return {
    id = id,
    name = name,
    kind = kind,
    tags = tags,
    dir = case_dir,
    path = case_dir,
    expected_dir = expected_dir,
    actual_dir = actual_dir,
    diff_dir = diff_dir,
    expected_path = util.normalize_path(expected_dir, "snapshot.json"),
    actual_path = util.normalize_path(actual_dir, "snapshot.json"),
    scenario_path = scenario,
    golden_scenario_path = golden,
    target_scenario_path = target,
  }
end

local function matches_tag(case_tags, filter_tags)
  if #filter_tags == 0 then
    return true
  end
  local tag_map = {}
  for _, tag in ipairs(case_tags) do
    tag_map[tag] = true
  end
  for _, tag in ipairs(filter_tags) do
    if tag_map[tag] then
      return true
    end
  end
  return false
end

local function matches_case_id(id, filter_ids)
  if #filter_ids == 0 then
    return true
  end
  for _, value in ipairs(filter_ids) do
    if value == id then
      return true
    end
  end
  return false
end

function M.filter_cases(cases, filter)
  local out = {}
  for _, c in ipairs(cases) do
    if matches_tag(c.tags, filter.tags or {}) and matches_case_id(c.id, filter.ids or {}) then
      table.insert(out, c)
    end
  end
  return out
end

function M.find_cases(root)
  local paths = vim.fn.globpath(root, "**/case.json", true, true)
  local cases = {}
  local errors = {}
  for _, path in ipairs(paths) do
    local c, err = M.load_case(path)
    if not c then
      table.insert(errors, string.format("%s: %s", path, err))
    else
      table.insert(cases, c)
    end
  end
  table.sort(cases, function(a, b)
    return a.id < b.id
  end)
  return cases, errors
end

function M.ensure_dir(path)
  if not path then
    return false, "path is required"
  end
  local ok = vim.fn.mkdir(path, "p")
  if ok == 0 then
    return false, "failed to create dir: " .. path
  end
  return true
end

return M

end)
__bundle_register("snap.command_suite_compare", function(require, _LOADED, __bundle_register, __bundle_modules)
local case_def = require("snap.case_def")
local normalize = require("snap.normalize")
local output = require("snap.output")
local png = require("snap.png")
local render = require("snap.render")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua compare [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case ID         Filter by case id (repeatable, comma-separated)",
    "  --format FMT      Diff formats: text,ansi,html,png (default: text)",
    "  --diff-always     Always generate diffs",
    "  --json            Output JSON summary",
    "  -h, --help        Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_format(value)
  local formats = {}
  for _, item in ipairs(split_values(value)) do
    formats[item] = true
  end
  if not next(formats) then
    formats.text = true
  end
  return formats
end

local function parse_args(args)
  local opts = {
    root = ".",
    tags = {},
    cases = {},
    formats = { text = true },
    diff_always = false,
    json = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.formats = parse_format(value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.formats = parse_format(string.sub(arg, 10))
    elseif arg == "--diff-always" then
      opts.diff_always = true
    elseif arg == "--json" then
      opts.json = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function read_input(path)
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

local function decode_json(text)
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    return nil, "failed to parse json: " .. tostring(decoded)
  end
  return decoded
end

local function deep_equal(a, b)
  if type(a) ~= type(b) then
    return false
  end
  if type(a) ~= "table" then
    return a == b
  end
  local seen = {}
  for key, value in pairs(a) do
    if not deep_equal(value, b[key]) then
      return false
    end
    seen[key] = true
  end
  for key, _ in pairs(b) do
    if not seen[key] then
      return false
    end
  end
  return true
end

local function escape_html(text)
  return (text:gsub("[&<>\"']", {
    ["&"] = "&amp;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ['"'] = "&quot;",
    ["'"] = "&#39;",
  }))
end

local function highlight_unified(diff_text)
  local out = { "<pre>" }
  for line in diff_text:gmatch("([^\n]*)\n?") do
    if line == "" and #out > 1 and out[#out] == "" then
      break
    end
    local cls = "line"
    if vim.startswith(line, "+") then
      cls = cls .. " add"
    elseif vim.startswith(line, "-") then
      cls = cls .. " del"
    elseif vim.startswith(line, "@@") then
      cls = cls .. " hunk"
    end
    table.insert(out, '<div class="' .. cls .. '">' .. escape_html(line) .. "</div>")
  end
  table.insert(out, "</pre>")
  return table.concat(out)
end

local function grid_text_matrix(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return 0, 0, {}
  end
  local rows = grid.rows or 0
  local cols = grid.cols or 0
  local matrix = {}
  for r = 1, rows do
    local row_cells = grid.cells[r] or {}
    local line = {}
    for c = 1, cols do
      local cell = row_cells[c] or { text = " " }
      local text = cell.text
      if text == "" then
        text = " "
      end
      line[c] = text
    end
    matrix[r] = line
  end
  return rows, cols, matrix
end

local function lines_from_matrix(rows, cols, matrix)
  local lines = {}
  for r = 1, rows do
    local row = matrix[r] or {}
    lines[r] = table.concat(row, "")
  end
  return lines
end

local function align_lines(expected_lines, actual_lines)
  local expected_text = table.concat(expected_lines, "\n")
  local actual_text = table.concat(actual_lines, "\n")
  local diffs = vim.text.diff(expected_text, actual_text, {
    result_type = "indices",
    algorithm = "patience",
    linematch = true,
    indent_heuristic = true,
  })
  local pairs = {}
  local e = 1
  local a = 1
  for _, d in ipairs(diffs) do
    local a_start, a_count, b_start, b_count = d[1], d[2], d[3], d[4]
    local expected_unchanged = (a_count == 0) and (a_start - e + 1) or (a_start - e)
    local actual_unchanged = (b_count == 0) and (b_start - a + 1) or (b_start - a)
    if expected_unchanged < 0 then
      expected_unchanged = 0
    end
    if actual_unchanged < 0 then
      actual_unchanged = 0
    end
    local common = math.min(expected_unchanged, actual_unchanged)
    for i = 0, common - 1 do
      table.insert(pairs, { e = e + i, a = a + i, kind = nil })
    end
    if expected_unchanged > common then
      for i = 0, expected_unchanged - common - 1 do
        table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
      end
    elseif actual_unchanged > common then
      for i = 0, actual_unchanged - common - 1 do
        table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
      end
    end
    local e_change = (a_count == 0) and (a_start + 1) or a_start
    local a_change = (b_count == 0) and (b_start + 1) or b_start
    e = e_change
    a = a_change
    local maxc = math.max(a_count, b_count)
    for i = 0, maxc - 1 do
      local er = (i < a_count) and (e + i) or 0
      local ar = (i < b_count) and (a + i) or 0
      local kind = nil
      if er == 0 and ar > 0 then
        kind = "added"
      elseif ar == 0 and er > 0 then
        kind = "removed"
      else
        kind = "changed"
      end
      table.insert(pairs, { e = er, a = ar, kind = kind })
    end
    e = e + a_count
    a = a + b_count
  end
  local expected_remaining = math.max(#expected_lines - e + 1, 0)
  local actual_remaining = math.max(#actual_lines - a + 1, 0)
  local common = math.min(expected_remaining, actual_remaining)
  for i = 0, common - 1 do
    table.insert(pairs, { e = e + i, a = a + i, kind = nil })
  end
  if expected_remaining > common then
    for i = 0, expected_remaining - common - 1 do
      table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
    end
  elseif actual_remaining > common then
    for i = 0, actual_remaining - common - 1 do
      table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
    end
  end
  return pairs
end

local function build_aligned_maps(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local expected_lines = lines_from_matrix(erows, ecols, ematrix)
  local actual_lines = lines_from_matrix(arows, acols, amatrix)
  local pairs = align_lines(expected_lines, actual_lines)
  local cols = math.max(ecols, acols)
  local expected_rows = {}
  local actual_rows = {}
  local expected_line_kinds = {}
  local actual_line_kinds = {}
  local expected_cells = {}
  local actual_cells = {}
  for idx, pair in ipairs(pairs) do
    expected_rows[idx] = pair.e
    actual_rows[idx] = pair.a
    if pair.kind == "removed" then
      expected_line_kinds[idx] = "removed"
    elseif pair.kind == "added" then
      actual_line_kinds[idx] = "added"
    elseif pair.kind == "changed" then
      expected_line_kinds[idx] = "removed"
      actual_line_kinds[idx] = "added"
    end
    if pair.e and pair.e > 0 and pair.a and pair.a > 0 then
      for c = 1, cols do
        local etext = ematrix[pair.e] and ematrix[pair.e][c] or " "
        local atext = amatrix[pair.a] and amatrix[pair.a][c] or " "
        if etext ~= atext then
          expected_cells[pair.e] = expected_cells[pair.e] or {}
          actual_cells[pair.a] = actual_cells[pair.a] or {}
          expected_cells[pair.e][c] = true
          actual_cells[pair.a][c] = true
        end
      end
    end
  end
  return {
    expected_rows = expected_rows,
    actual_rows = actual_rows,
    expected_line_kinds = expected_line_kinds,
    actual_line_kinds = actual_line_kinds,
    expected_cells = expected_cells,
    actual_cells = actual_cells,
  }
end

local function build_diff_map(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local rows = math.max(erows, arows)
  local cols = math.max(ecols, acols)
  local expected = { lines = {}, cells = {} }
  local actual = { lines = {}, cells = {} }
  for r = 1, rows do
    local line_diff = false
    for c = 1, cols do
      local etext = ematrix[r] and ematrix[r][c] or " "
      local atext = amatrix[r] and amatrix[r][c] or " "
      if etext ~= atext then
        expected.cells[r] = expected.cells[r] or {}
        actual.cells[r] = actual.cells[r] or {}
        expected.cells[r][c] = true
        actual.cells[r][c] = true
        line_diff = true
      end
    end
    if line_diff then
      expected.lines[r] = true
      actual.lines[r] = true
    end
  end
  return { expected = expected, actual = actual }
end

local function wrap_html_diff(unified_diff, expected_plain, actual_plain, expected_aligned, actual_aligned, default_view)
  if default_view ~= "side" and default_view ~= "overlay" then
    default_view = "unified"
  end
  local unified_checked = default_view == "unified" and " checked" or ""
  local side_checked = default_view == "side" and " checked" or ""
  local overlay_checked = default_view == "overlay" and " checked" or ""
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>nvim-snap compare</title>",
    "  <style>",
    "    :root { color-scheme: light; }",
    "    body { margin: 0; background: #f4f5f7; color: #1f2328; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }",
    "    .wrap { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; padding: 0; }",
    "    .panel { background: #ffffff; border: 1px solid #d0d7de; border-radius: 6px; overflow: auto; }",
    "    .title { padding: 8px 12px; border-bottom: 1px solid #222; font-weight: 600; }",
    "    .content { padding: 12px; }",
    "    .content pre { margin: 0; white-space: pre; }",
    "    .section { padding: 12px 16px 16px; }",
    "    .tabs { display: inline-flex; gap: 4px; padding: 10px 12px; margin: 12px 16px 0; align-items: center; background: #ffffff; border: 1px solid #d0d7de; border-radius: 10px; box-shadow: 0 1px 2px rgba(16, 24, 40, 0.08); }",
    "    .tabs .label { color: #57606a; font-size: 1em; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; padding: 4px 8px; border: 1px solid #e1e4e8; border-radius: 999px; background: #f6f8fa; }",
    "    .tabs label { color: #24292f; padding: 6px 12px; border-radius: 8px; border: 1px solid transparent; cursor: pointer; user-select: none; }",
    "    .tabs label:hover { background: #f6f8fa; }",
    "    .toggles { display: none; }",
    "    #view-unified:checked ~ .tabs label[for=\"view-unified\"],",
    "    #view-side:checked ~ .tabs label[for=\"view-side\"],",
    "    #view-side-diff:checked ~ .tabs label[for=\"view-side-diff\"] {",
    "      color: #ffffff; background: #2da44e; border-color: #2da44e; }",
    "    .cell.diff.added { background: rgba(34,197,94,0.28); box-shadow: inset 0 0 0 1px rgba(34,197,94,0.7); }",
    "    .cell.diff.removed { background: rgba(220,38,38,0.22); box-shadow: inset 0 0 0 1px rgba(220,38,38,0.7); }",
    "    .view-plain .line.diff, .view-plain .line.diff .cell, .view-plain .cell.diff { background: none; box-shadow: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.added .cell { box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.removed .cell { box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.added {",
    "      box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.45), inset 0 0 0 1px rgba(34,197,94,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(34,197,94,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.removed {",
    "      box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.4), inset 0 0 0 1px rgba(220,38,38,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(220,38,38,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    .udiff { color: #24292f; }",
    "    .udiff .line { padding: 0 4px; }",
    "    .udiff .line.add { background: rgba(34,197,94,0.18); }",
    "    .udiff .line.del { background: rgba(220,38,38,0.18); }",
    "    .udiff .line.hunk { color: #0969da; }",
    "    #view-unified:not(:checked) ~ .page #unified { display: none; }",
    "    #view-unified:checked ~ .page #side { display: none; }",
    "    #view-side:checked ~ .page #side { display: block; }",
    "    #view-side-diff:checked ~ .page #side { display: block; }",
    "    #view-side:checked ~ .page #side .view-plain { display: block; }",
    "    #view-side:checked ~ .page #side .view-aligned { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-plain { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned { display: block; }",
    "    .grid { display: inline-block; }",
    "    .line { display: block; white-space: pre; }",
    "    .cell { display: inline-block; }",
    "  </style>",
    "</head>",
    "<body>",
    "  <input id=\"view-unified\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. unified_checked .. " />",
    "  <input id=\"view-side\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. side_checked .. " />",
    "  <input id=\"view-side-diff\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. overlay_checked .. " />",
    "  <div class=\"tabs\">",
    "    <span class=\"label\">view</span>",
    "    <label for=\"view-unified\">unified</label>",
    "    <label for=\"view-side\">side</label>",
    "    <label for=\"view-side-diff\">overlay</label>",
    "  </div>",
    "  <div class=\"page\">",
    "  <div class=\"section\" id=\"unified\">",
    "    <div class=\"panel\">",
    "      <div class=\"title\">unified diff (text)</div>",
    "      <div class=\"content udiff\">" .. unified_diff .. "</div>",
    "    </div>",
    "  </div>",
    "  <div class=\"section\" id=\"side\">",
    "    <div class=\"wrap\">",
    "      <div class=\"panel\">",
    "        <div class=\"title\">expected</div>",
    "        <div class=\"content view-plain\" style=\"background:" .. expected_plain.bg .. ";color:" .. expected_plain.fg .. ";\"><div class=\"grid\">" .. expected_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. expected_aligned.bg .. ";color:" .. expected_aligned.fg .. ";\"><div class=\"grid\">" .. expected_aligned.html .. "</div></div>",
    "      </div>",
    "      <div class=\"panel\">",
    "        <div class=\"title\">actual</div>",
    "        <div class=\"content view-plain\" style=\"background:" .. actual_plain.bg .. ";color:" .. actual_plain.fg .. ";\"><div class=\"grid\">" .. actual_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. actual_aligned.bg .. ";color:" .. actual_aligned.fg .. ";\"><div class=\"grid\">" .. actual_aligned.html .. "</div></div>",
    "      </div>",
    "    </div>",
    "  </div>",
    "  </div>",
    "</body>",
    "</html>",
  }, "\n")
end

local function render_html_diff(expected, actual, default_view)
  local expected_render_text = render.render_text(expected)
  local actual_render_text = render.render_text(actual)
  local unified = vim.text.diff(expected_render_text, actual_render_text, { result_type = "unified", ctxlen = 3 })
  local diff_map = build_diff_map(expected, actual)
  local expected_plain = render.render_html_cells(expected, diff_map.expected, "removed")
  local actual_plain = render.render_html_cells(actual, diff_map.actual, "added")
  local aligned = build_aligned_maps(expected, actual)
  local expected_aligned = render.render_html_aligned(
    expected,
    aligned.expected_rows,
    aligned.expected_line_kinds,
    aligned.expected_cells,
    "removed"
  )
  local actual_aligned = render.render_html_aligned(
    actual,
    aligned.actual_rows,
    aligned.actual_line_kinds,
    aligned.actual_cells,
    "added"
  )
  return wrap_html_diff(
    highlight_unified(unified or ""),
    expected_plain,
    actual_plain,
    expected_aligned,
    actual_aligned,
    default_view
  )
end

local function diff_files(expected, actual, formats)
  local results = {}
  if formats.text then
    local diff = vim.text.diff(render.render_text(expected), render.render_text(actual), {
      result_type = "unified",
      ctxlen = 3,
    })
    results.text = diff or ""
  end
  if formats.ansi then
    local diff = vim.text.diff(render.render_ansi(expected), render.render_ansi(actual), {
      result_type = "unified",
      ctxlen = 3,
    })
    results.ansi = diff or ""
  end
  if formats.html then
    results.html = render_html_diff(expected, actual, "unified")
  end
  if formats.png then
    results.png = render_html_diff(expected, actual, "overlay")
  end
  return results
end

local function write_diff_outputs(c, outputs)
  local ok, err = case_def.ensure_dir(c.diff_dir)
  if not ok then
    return nil, err
  end
  local diff_paths = {}
  if outputs.text ~= nil then
    local path = vim.fs.joinpath(c.diff_dir, "diff.txt")
    local ok_write, write_err = output.write(path, outputs.text)
    if not ok_write then
      return nil, write_err or "failed to write diff.txt"
    end
    diff_paths.text = path
  end
  if outputs.ansi ~= nil then
    local path = vim.fs.joinpath(c.diff_dir, "diff.ansi")
    local ok_write, write_err = output.write(path, outputs.ansi)
    if not ok_write then
      return nil, write_err or "failed to write diff.ansi"
    end
    diff_paths.ansi = path
  end
  if outputs.html ~= nil then
    local path = vim.fs.joinpath(c.diff_dir, "diff.html")
    local ok_write, write_err = output.write(path, outputs.html)
    if not ok_write then
      return nil, write_err or "failed to write diff.html"
    end
    diff_paths.html = path
  end
  if outputs.png ~= nil then
    local path = vim.fs.joinpath(c.diff_dir, "diff.png")
    local ok_write, write_err = png.write_png_from_html(outputs.png, path)
    if not ok_write then
      return nil, write_err or "failed to write diff.png"
    end
    diff_paths.png = path
  end
  return diff_paths
end

local function print_text(results)
  for _, r in ipairs(results) do
    local tags = table.concat(r.tags, ",")
    local diff_paths = ""
    if r.diff_paths then
      local items = {}
      for key, value in pairs(r.diff_paths) do
        table.insert(items, key .. "=" .. value)
      end
      table.sort(items)
      diff_paths = table.concat(items, ",")
    end
    print(table.concat({
      r.id,
      r.name,
      r.kind,
      tags,
      r.result,
      diff_paths,
      r.error_reason or "",
    }, "\t"))
  end
end

local function print_json(root, results, summary)
  local out = {
    root = root,
    summary = summary,
    cases = results,
  }
  print(vim.json.encode(out, { indent = "  " }))
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases, errors = case_def.find_cases(root)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })
  local results = {}
  local summary = { total = 0, no_diff = 0, diff = 0, error = 0 }
  local has_error = #errors > 0
  local has_diff = false

  for _, err in ipairs(errors) do
    util.err_write(err)
  end

  for _, c in ipairs(filtered) do
    summary.total = summary.total + 1
    local entry = {
      id = c.id,
      name = c.name,
      kind = c.kind,
      tags = c.tags,
      result = "error",
      diff_paths = nil,
      error_reason = nil,
    }
    local expected_text, expected_err = read_input(c.expected_path)
    if not expected_text then
      entry.error_reason = expected_err or "expected not found"
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.id .. ": " .. entry.error_reason)
      goto continue
    end
    local actual_text, actual_err = read_input(c.actual_path)
    if not actual_text then
      entry.error_reason = actual_err or "actual not found"
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.id .. ": " .. entry.error_reason)
      goto continue
    end
    local expected_snapshot, expected_decode_err = decode_json(expected_text)
    if not expected_snapshot then
      entry.error_reason = expected_decode_err
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.id .. ": " .. entry.error_reason)
      goto continue
    end
    local actual_snapshot, actual_decode_err = decode_json(actual_text)
    if not actual_snapshot then
      entry.error_reason = actual_decode_err
      summary.error = summary.error + 1
      has_error = true
      table.insert(results, entry)
      util.err_write(c.id .. ": " .. entry.error_reason)
      goto continue
    end

    local normalized_expected = normalize.normalize(expected_snapshot)
    local normalized_actual = normalize.normalize(actual_snapshot)
    local equal = deep_equal(normalized_expected, normalized_actual)
    if equal then
      entry.result = "no_diff"
      summary.no_diff = summary.no_diff + 1
    else
      entry.result = "diff"
      summary.diff = summary.diff + 1
      has_diff = true
    end

    if entry.result == "diff" or opts.diff_always then
      local outputs = diff_files(normalized_expected, normalized_actual, opts.formats)
      local diff_paths, write_err = write_diff_outputs(c, outputs)
      if not diff_paths then
        entry.result = "error"
        entry.error_reason = write_err or "failed to write diff"
        summary.error = summary.error + 1
        has_error = true
        util.err_write(c.id .. ": " .. entry.error_reason)
      else
        entry.diff_paths = diff_paths
      end
    end

    table.insert(results, entry)
    ::continue::
  end

  if opts.json then
    print_json(root, results, summary)
  else
    print_text(results)
  end

  if has_error then
    vim.cmd("cquit 2")
  elseif has_diff then
    vim.cmd("cquit 1")
  end
end

return M

end)
__bundle_register("snap.render", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

function M.render_text(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return ""
  end

  local out = {}
  for r = 1, grid.rows do
    local row_cells = grid.cells[r] or {}
    local line = {}
    for c = 1, grid.cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      table.insert(line, text)
    end
    table.insert(out, table.concat(line))
  end
  return table.concat(out, "\n")
end

local function build_attr_map(snapshot)
  local default_fg = snapshot.default_colors and snapshot.default_colors.rgb_fg or nil
  local default_bg = snapshot.default_colors and snapshot.default_colors.rgb_bg or nil
  local attr_map = {}
  for _, attr in ipairs(snapshot.hl_attrs or {}) do
    attr_map[attr.id] = attr.rgb_attr or {}
  end
  return attr_map, default_fg, default_bg
end

local function style_from(attr_map, default_fg, default_bg, hl_id)
  local attr = attr_map[hl_id] or {}
  local fg = attr.foreground
  local bg = attr.background
  local reverse = attr.reverse == true
  if fg == nil then
    fg = default_fg
  end
  if bg == nil then
    bg = default_bg
  end
  if reverse then
    fg, bg = bg, fg
  end
  return {
    fg = fg,
    bg = bg,
    bold = attr.bold == true,
    italic = attr.italic == true,
    underline = attr.underline == true
      or attr.undercurl == true
      or attr.underdouble == true
      or attr.underdotted == true
      or attr.underdashed == true,
    strikethrough = attr.strikethrough == true,
    reverse = reverse,
  }
end

local function to_hex_color(color)
  if color == nil or color == vim.NIL then
    return nil
  end
  if type(color) ~= "number" or color < 0 then
    return nil
  end
  return string.format("#%06x", color)
end

local function style_to_css(style)
  local parts = {}
  local fg = to_hex_color(style.fg)
  local bg = to_hex_color(style.bg)
  if fg then
    table.insert(parts, "color:" .. fg)
  end
  if bg then
    table.insert(parts, "background-color:" .. bg)
  end
  if style.bold then
    table.insert(parts, "font-weight:700")
  end
  if style.italic then
    table.insert(parts, "font-style:italic")
  end
  local decorations = {}
  if style.underline then
    table.insert(decorations, "underline")
  end
  if style.strikethrough then
    table.insert(decorations, "line-through")
  end
  if #decorations > 0 then
    table.insert(parts, "text-decoration:" .. table.concat(decorations, " "))
  end
  return table.concat(parts, ";")
end

local function rgb_to_ansi(color, is_bg)
  if color == nil or color == vim.NIL then
    return nil
  end
  if type(color) ~= "number" or color < 0 then
    return nil
  end
  local r = math.floor(color / 65536) % 256
  local g = math.floor(color / 256) % 256
  local b = color % 256
  return string.format("\x1b[%d;2;%d;%d;%dm", is_bg and 48 or 38, r, g, b)
end

function M.render_ansi(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return ""
  end

  local attr_map, default_fg, default_bg = build_attr_map(snapshot)

  local function to_style(hl_id)
    return style_from(attr_map, default_fg, default_bg, hl_id)
  end

  local function style_equal(a, b)
    return a.fg == b.fg
      and a.bg == b.bg
      and a.bold == b.bold
      and a.italic == b.italic
      and a.underline == b.underline
      and a.strikethrough == b.strikethrough
      and a.reverse == b.reverse
  end

  local function style_to_ansi(style)
    local codes = { "\x1b[0m" }
    if style.bold then
      table.insert(codes, "\x1b[1m")
    end
    if style.italic then
      table.insert(codes, "\x1b[3m")
    end
    if style.underline then
      table.insert(codes, "\x1b[4m")
    end
    if style.strikethrough then
      table.insert(codes, "\x1b[9m")
    end
    local fg = rgb_to_ansi(style.fg, false)
    local bg = rgb_to_ansi(style.bg, true)
    if fg then
      table.insert(codes, fg)
    end
    if bg then
      table.insert(codes, bg)
    end
    return table.concat(codes)
  end

  local out = {}
  for r = 1, grid.rows do
    local row_cells = grid.cells[r] or {}
    local current = {
      fg = nil,
      bg = nil,
      bold = false,
      italic = false,
      underline = false,
      strikethrough = false,
      reverse = false,
    }
    local line = {}
    for c = 1, grid.cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      local style = to_style(cell.hl_id or 0)
      if not style_equal(style, current) then
        table.insert(line, style_to_ansi(style))
        current = style
      end
      table.insert(line, text)
    end
    table.insert(line, "\x1b[0m")
    table.insert(out, table.concat(line))
  end
  return table.concat(out, "\n")
end

local function escape_html(text)
  return (text:gsub("[&<>\"']", {
    ["&"] = "&amp;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ['"'] = "&quot;",
    ["'"] = "&#39;",
  }))
end

function M.render_html_fragment(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return ""
  end

  local attr_map, default_fg, default_bg = build_attr_map(snapshot)

  local function to_style(hl_id)
    local style = style_from(attr_map, default_fg, default_bg, hl_id)
    style.reverse = nil
    return style
  end

  local function style_equal(a, b)
    return a.fg == b.fg
      and a.bg == b.bg
      and a.bold == b.bold
      and a.italic == b.italic
      and a.underline == b.underline
      and a.strikethrough == b.strikethrough
  end

  local lines = {}
  for r = 1, grid.rows do
    local row_cells = grid.cells[r] or {}
    local current = {
      fg = nil,
      bg = nil,
      bold = false,
      italic = false,
      underline = false,
      strikethrough = false,
    }
    local line = {}
    local chunk = {}
    for c = 1, grid.cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      local style = to_style(cell.hl_id or 0)
      if not style_equal(style, current) then
        if #chunk > 0 then
          local css = style_to_css(current)
          if css ~= "" then
            table.insert(line, '<span style="' .. css .. '">' .. escape_html(table.concat(chunk)) .. "</span>")
          else
            table.insert(line, escape_html(table.concat(chunk)))
          end
          chunk = {}
        end
        current = style
      end
      table.insert(chunk, text)
    end
    if #chunk > 0 then
      local css = style_to_css(current)
      if css ~= "" then
        table.insert(line, '<span style="' .. css .. '">' .. escape_html(table.concat(chunk)) .. "</span>")
      else
        table.insert(line, escape_html(table.concat(chunk)))
      end
    end
    table.insert(lines, table.concat(line))
  end

  local bg = to_hex_color(default_bg) or "#000000"
  local fg = to_hex_color(default_fg) or "#ffffff"
  return {
    bg = bg,
    fg = fg,
    html = "<pre>" .. table.concat(lines, "\n") .. "</pre>",
  }
end

function M.render_html_aligned(snapshot, row_indices, line_kinds, cell_diff, diff_kind)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return { bg = "#000000", fg = "#ffffff", html = "" }
  end

  local attr_map, default_fg, default_bg = build_attr_map(snapshot)
  local bg = to_hex_color(default_bg) or "#000000"
  local fg = to_hex_color(default_fg) or "#ffffff"
  local cols = grid.cols or 0

  local lines = {}
  for idx, row_index in ipairs(row_indices) do
    local row_cells = {}
    if row_index and row_index > 0 then
      row_cells = grid.cells[row_index] or {}
    end
    local line_classes = { "line" }
    local kind = line_kinds and line_kinds[idx] or nil
    if kind then
      table.insert(line_classes, "diff")
      table.insert(line_classes, kind)
    end
    local line = { '<div class="' .. table.concat(line_classes, " ") .. '">' }
    for c = 1, cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      local style = style_from(attr_map, default_fg, default_bg, cell.hl_id or 0)
      local css = style_to_css(style)
      local cell_classes = { "cell" }
      if row_index and row_index > 0 and cell_diff and cell_diff[row_index] and cell_diff[row_index][c] then
        table.insert(cell_classes, "diff")
        table.insert(cell_classes, diff_kind)
      end
      local open = '<span class="' .. table.concat(cell_classes, " ") .. '"'
      if css ~= "" then
        open = open .. ' style="' .. css .. '"'
      end
      open = open .. ">"
      table.insert(line, open .. escape_html(text) .. "</span>")
    end
    table.insert(line, "</div>")
    table.insert(lines, table.concat(line))
  end

  return {
    bg = bg,
    fg = fg,
    html = table.concat(lines),
  }
end

function M.render_html_cells(snapshot, diff_map, diff_kind)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return { bg = "#000000", fg = "#ffffff", html = "" }
  end
  local row_indices = {}
  local line_kinds = {}
  for r = 1, grid.rows do
    row_indices[r] = r
    if diff_map and diff_map.lines and diff_map.lines[r] then
      line_kinds[r] = diff_kind
    end
  end
  return M.render_html_aligned(snapshot, row_indices, line_kinds, diff_map and diff_map.cells or nil, diff_kind)
end

function M.render_html(snapshot)
  local fragment = M.render_html_fragment(snapshot)
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>Neovim UI Snapshot</title>",
    "  <style>",
    "    body {",
    "      margin: 0;",
    "      background: " .. fragment.bg .. ";",
    "      color: " .. fragment.fg .. ";",
    "      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;",
    "      line-height: 1.2;",
    "    }",
    "    pre {",
    "      margin: 0;",
    "      padding: 16px;",
    "      white-space: pre;",
    "    }",
    "  </style>",
    "</head>",
    "<body>",
    "  " .. fragment.html,
    "</body>",
    "</html>",
  }, "\n")
end

return M

end)
__bundle_register("snap.png", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

local function parse_size()
  local value = vim.env.SNAP_PNG_SIZE
  if type(value) ~= "string" or value == "" then
    return 1600, 1200
  end
  local w, h = value:match("^(%d+)[xX](%d+)$")
  w = tonumber(w)
  h = tonumber(h)
  if not w or not h or w <= 0 or h <= 0 then
    return 1600, 1200
  end
  return w, h
end

local function find_tool()
  local candidates = {
    "google-chrome",
    "chrome",
    "msedge",
    "chromium",
    "chromium-browser",
    "wkhtmltoimage",
  }
  for _, cmd in ipairs(candidates) do
    if vim.fn.executable(cmd) == 1 then
      return cmd
    end
  end
  return nil
end

local function write_file(path, contents)
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

local function run_chromium(cmd, html_path, out_path, width, height, user_data_dir)
  local url = vim.uri_from_fname(html_path)
  local profile_dir = vim.fs.joinpath(user_data_dir, "profile")
  vim.fn.mkdir(profile_dir, "p")
  local args = {
    "env",
    "HOME=" .. user_data_dir,
    "XDG_DATA_HOME=" .. user_data_dir,
    "XDG_CONFIG_HOME=" .. user_data_dir,
    cmd,
    "--headless",
    "--disable-gpu",
    "--no-sandbox",
    "--no-first-run",
    "--disable-extensions",
    "--disable-dev-shm-usage",
    "--disable-crash-reporter",
    "--disable-breakpad",
    "--crash-dumps-dir=" .. user_data_dir,
    "--disable-features=Translate,Crashpad",
    "--user-data-dir=" .. profile_dir,
    "--window-size=" .. width .. "," .. height,
    "--screenshot=" .. out_path,
    url,
  }
  local output = vim.fn.system(args)
  if vim.v.shell_error ~= 0 then
    return nil, output
  end
  return true
end

local function run_wkhtmltoimage(cmd, html_path, out_path, width)
  local args = {
    cmd,
    "--width",
    tostring(width),
    "--disable-smart-width",
    html_path,
    out_path,
  }
  local output = vim.fn.system(args)
  if vim.v.shell_error ~= 0 then
    return nil, output
  end
  return true
end

function M.write_png_from_html(html, out_path)
  local tool = find_tool()
  if not tool then
    return nil, "png tool not found (chromium/chrome/msedge/wkhtmltoimage)"
  end
  local width, height = parse_size()
  local tmp = vim.fn.tempname() .. ".html"
  local user_data_dir = vim.fn.tempname()
  local ok, err = write_file(tmp, html)
  if not ok then
    return nil, err
  end
  if tool ~= "wkhtmltoimage" then
    vim.fn.mkdir(user_data_dir, "p")
  end
  local ok_run, run_err
  if tool == "wkhtmltoimage" then
    ok_run, run_err = run_wkhtmltoimage(tool, tmp, out_path, width)
  else
    ok_run, run_err = run_chromium(tool, tmp, out_path, width, height, user_data_dir)
  end
  os.remove(tmp)
  if tool ~= "wkhtmltoimage" then
    vim.fn.delete(user_data_dir, "rf")
  end
  if not ok_run then
    return nil, run_err
  end
  return true
end

return M

end)
__bundle_register("snap.normalize", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

local function normalize_list(value, key_field)
  if type(value) ~= "table" then
    return {}
  end
  local list = {}
  for key, item in pairs(value) do
    if type(item) == "table" then
      local entry = item
      if entry[key_field] == nil and type(key) == "number" then
        entry = vim.deepcopy(entry)
        entry[key_field] = key
      end
      table.insert(list, entry)
    end
  end
  table.sort(list, function(a, b)
    return (a[key_field] or 0) < (b[key_field] or 0)
  end)
  return list
end

local function normalize_groups(value)
  if type(value) ~= "table" then
    return {}
  end
  local list = {}
  for _, item in pairs(value) do
    if type(item) == "table" then
      table.insert(list, item)
    end
  end
  table.sort(list, function(a, b)
    return tostring(a.name or "") < tostring(b.name or "")
  end)
  return list
end

function M.normalize(snapshot)
  if type(snapshot) ~= "table" then
    return {}
  end
  local normalized = {}
  for key, value in pairs(snapshot) do
    normalized[key] = value
  end

  normalized.grids = normalize_list(snapshot.grids, "id")
  normalized.hl_attrs = normalize_list(snapshot.hl_attrs, "id")
  normalized.hl_groups = normalize_groups(snapshot.hl_groups)

  return normalized
end

return M

end)
__bundle_register("snap.command_run", function(require, _LOADED, __bundle_register, __bundle_modules)
local case_def = require("snap.case_def")
local output = require("snap.output")
local render = require("snap.render")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua run [options]",
    "",
    "options:",
    "  --root PATH       Root directory to search (default: .)",
    "  --tag TAG         Filter by tag (repeatable, comma-separated)",
    "  --case ID         Filter by case id (repeatable, comma-separated)",
    "  --format FMT      Output formats: json,ansi,html (default: json)",
    "  -h, --help        Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_format(value)
  local formats = {}
  for _, item in ipairs(split_values(value)) do
    formats[item] = true
  end
  if not next(formats) then
    formats.json = true
  end
  return formats
end

local function parse_args(args)
  local opts = {
    root = ".",
    tags = {},
    cases = {},
    formats = { json = true },
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.formats = parse_format(value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.formats = parse_format(string.sub(arg, 10))
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function scenario_for_case(c)
  if c.kind == "regression" then
    return c.scenario_path
  end
  return c.target_scenario_path
end

local function collect_snapshot(c)
  local scenario = scenario_for_case(c)
  if vim.fn.filereadable(scenario) ~= 1 then
    return nil, "scenario not found: " .. scenario
  end
  local opts = {
    scripts = { scenario },
    width = 80,
    height = 24,
    wait = 200,
    rpc_timeout = 2000,
    nvim = "nvim",
    data_home = util.normalize_path(c.dir, ".nvim-data"),
    config_home = util.normalize_path(c.dir, ".nvim-config"),
    multigrid = false,
  }
  return snapshot.collect(opts)
end

local function write_outputs(c, formats, snap)
  local ok, err = case_def.ensure_dir(c.actual_dir)
  if not ok then
    return nil, err
  end
  if formats.json then
    local encoded = vim.json.encode(snap)
    local ok_write, write_err = output.write(c.actual_path, encoded)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.json"
    end
  end
  if formats.ansi then
    local ansi = render.render_ansi(snap)
    local ok_write, write_err = output.write(vim.fs.joinpath(c.actual_dir, "snapshot.ansi"), ansi)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.ansi"
    end
  end
  if formats.html then
    local html = render.render_html(snap)
    local ok_write, write_err = output.write(vim.fs.joinpath(c.actual_dir, "snapshot.html"), html)
    if not ok_write then
      return nil, write_err or "failed to write snapshot.html"
    end
  end
  return true
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases, errors = case_def.find_cases(root)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })
  local failed = #errors > 0

  for _, c in ipairs(filtered) do
    local snap, err = collect_snapshot(c)
    if not snap then
      util.err_write(c.id .. ": " .. (err or "capture failed"))
      failed = true
    else
      local ok_write, write_err = write_outputs(c, opts.formats, snap)
      if not ok_write then
        util.err_write(c.id .. ": " .. (write_err or "failed to write outputs"))
        failed = true
      end
    end
  end

  if failed then
    vim.cmd("cquit 1")
  end
end

return M

end)
__bundle_register("snap.command_new", function(require, _LOADED, __bundle_register, __bundle_modules)
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua new [options]",
    "",
    "options:",
    "  --root PATH       Root directory to create case (default: .)",
    "  --dir PATH        Case directory (overrides --root/--id)",
    "  --id ID           Case id (required unless --dir is used)",
    "  --name NAME       Case display name",
    "  --kind KIND       regression|golden (default: regression)",
    "  --tag TAG         Tag (repeatable, comma-separated)",
    "  --force           Overwrite existing files",
    "  -h, --help        Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_args(args)
  local opts = {
    root = ".",
    id = nil,
    dir = nil,
    name = nil,
    kind = "regression",
    tags = {},
    force = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--dir" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--dir requires a value")
      else
        opts.dir = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--dir=") then
      opts.dir = string.sub(arg, 7)
    elseif arg == "--id" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--id requires a value")
      else
        opts.id = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--id=") then
      opts.id = string.sub(arg, 6)
    elseif arg == "--name" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--name requires a value")
      else
        opts.name = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--name=") then
      opts.name = string.sub(arg, 8)
    elseif arg == "--kind" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--kind requires a value")
      else
        opts.kind = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--kind=") then
      opts.kind = string.sub(arg, 8)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--force" then
      opts.force = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function file_exists(path)
  return vim.fn.filereadable(path) == 1
end

local function write_file(path, contents, force)
  if file_exists(path) and not force then
    return nil, "file exists: " .. path
  end
  local dir = vim.fn.fnamemodify(path, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

local function write_case_json(path, opts)
  local payload = {
    version = 1,
    id = opts.id,
    kind = opts.kind,
  }
  if opts.name and opts.name ~= "" then
    payload.name = opts.name
  end
  if #opts.tags > 0 then
    payload.tags = opts.tags
  end
  local encoded = vim.json.encode(payload, { indent = "  " })
  return write_file(path, encoded, opts.force)
end

local function write_regression_scenario(path, id, force)
  local contents = table.concat({
    "vim.cmd(\"enew\")",
    "vim.fn.setline(1, {",
    "  \"case: " .. id .. "\",",
    "  \"edit this scenario\",",
    "})",
    "",
  }, "\n")
  return write_file(path, contents, force)
end

local function write_golden_scenario(path, id, label, force)
  local contents = table.concat({
    "vim.cmd(\"enew\")",
    "vim.fn.setline(1, {",
    "  \"" .. label .. " view for " .. id .. "\",",
    "  \"edit this scenario\",",
    "})",
    "",
  }, "\n")
  return write_file(path, contents, force)
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.kind ~= "regression" and opts.kind ~= "golden" then
    util.err_write("--kind must be regression or golden")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local case_dir
  if opts.dir and opts.dir ~= "" then
    case_dir = vim.fs.normalize(vim.fn.fnamemodify(opts.dir, ":p"))
    if not opts.id or opts.id == "" then
      opts.id = vim.fs.basename(case_dir)
    end
  else
    if not opts.id or opts.id == "" then
      util.err_write("--id is required")
      util.err_write(usage())
      vim.cmd("cq")
      return
    end
    case_dir = vim.fs.normalize(vim.fs.joinpath(root, opts.id))
  end

  if vim.fn.isdirectory(case_dir) == 1 and not opts.force then
    util.err_write("case directory already exists: " .. case_dir)
    vim.cmd("cq")
    return
  end
  vim.fn.mkdir(case_dir, "p")

  local ok, err = write_case_json(vim.fs.joinpath(case_dir, "case.json"), opts)
  if not ok then
    util.err_write(err or "failed to write case.json")
    vim.cmd("cq")
    return
  end

  vim.fn.mkdir(vim.fs.joinpath(case_dir, "expected"), "p")
  vim.fn.mkdir(vim.fs.joinpath(case_dir, "actual"), "p")
  vim.fn.mkdir(vim.fs.joinpath(case_dir, "diff"), "p")

  if opts.kind == "regression" then
    local ok_s, err_s = write_regression_scenario(vim.fs.joinpath(case_dir, "scenario.lua"), opts.id, opts.force)
    if not ok_s then
      util.err_write(err_s or "failed to write scenario.lua")
      vim.cmd("cq")
      return
    end
  else
    local ok_g, err_g = write_golden_scenario(vim.fs.joinpath(case_dir, "golden.lua"), opts.id, "golden", opts.force)
    if not ok_g then
      util.err_write(err_g or "failed to write golden.lua")
      vim.cmd("cq")
      return
    end
    local ok_t, err_t = write_golden_scenario(vim.fs.joinpath(case_dir, "target.lua"), opts.id, "target", opts.force)
    if not ok_t then
      util.err_write(err_t or "failed to write target.lua")
      vim.cmd("cq")
      return
    end
  end

  print(case_dir)
end

return M

end)
__bundle_register("snap.command_list", function(require, _LOADED, __bundle_register, __bundle_modules)
local case_def = require("snap.case_def")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua list [options]",
    "",
    "options:",
    "  --root PATH    Root directory to search (default: .)",
    "  --tag TAG      Filter by tag (repeatable, comma-separated)",
    "  --case ID      Filter by case id (repeatable, comma-separated)",
    "  --json         Output JSON",
    "  -h, --help     Show this help",
  }, "\n")
end

local function split_values(value)
  local out = {}
  for item in string.gmatch(value, "([^,]+)") do
    local trimmed = vim.trim(item)
    if trimmed ~= "" then
      table.insert(out, trimmed)
    end
  end
  return out
end

local function parse_args(args)
  local opts = {
    root = ".",
    tags = {},
    cases = {},
    json = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--tag" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--tag requires a value")
      else
        vim.list_extend(opts.tags, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--tag=") then
      vim.list_extend(opts.tags, split_values(string.sub(arg, 7)))
    elseif arg == "--case" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--case requires a value")
      else
        vim.list_extend(opts.cases, split_values(value))
        i = i + 1
      end
    elseif vim.startswith(arg, "--case=") then
      vim.list_extend(opts.cases, split_values(string.sub(arg, 8)))
    elseif arg == "--json" then
      opts.json = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function print_text(cases)
  for _, c in ipairs(cases) do
    local tags = table.concat(c.tags, ",")
    print(table.concat({ c.id, c.name, c.kind, tags, c.path }, "\t"))
  end
end

local function print_json(root, cases)
  local out = {
    root = root,
    cases = {},
  }
  for _, c in ipairs(cases) do
    table.insert(out.cases, {
      id = c.id,
      name = c.name,
      kind = c.kind,
      tags = c.tags,
      path = c.path,
    })
  end
  print(vim.json.encode(out, { indent = "  " }))
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local root = vim.fs.normalize(vim.fn.fnamemodify(opts.root, ":p"))
  local cases, errors = case_def.find_cases(root)
  local filtered = case_def.filter_cases(cases, { tags = opts.tags, ids = opts.cases })

  if #errors > 0 then
    for _, msg in ipairs(errors) do
      util.err_write(msg)
    end
  end

  if opts.json then
    print_json(root, filtered)
  else
    print_text(filtered)
  end

  if #errors > 0 then
    vim.cmd("cquit 1")
  end
end

return M

end)
__bundle_register("snap.command_ci", function(require, _LOADED, __bundle_register, __bundle_modules)
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua ci init [options]",
    "",
    "options:",
    "  --path PATH      Output workflow path (default: .github/workflows/nvim-snap.yml)",
    "  --root PATH      Root directory for cases (default: .)",
    "  --format FMT     Compare formats (default: html)",
    "  --name NAME      Workflow name (default: nvim-snap)",
    "  --force          Overwrite existing workflow file",
    "  -h, --help       Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    path = ".github/workflows/nvim-snap.yml",
    root = ".",
    format = "html",
    name = "nvim-snap",
    force = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--path" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--path requires a value")
      else
        opts.path = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--path=") then
      opts.path = string.sub(arg, 8)
    elseif arg == "--root" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--root requires a value")
      else
        opts.root = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--root=") then
      opts.root = string.sub(arg, 8)
    elseif arg == "--format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--format requires a value")
      else
        opts.format = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--format=") then
      opts.format = string.sub(arg, 10)
    elseif arg == "--name" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--name requires a value")
      else
        opts.name = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--name=") then
      opts.name = string.sub(arg, 8)
    elseif arg == "--force" then
      opts.force = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function workflow_yaml(opts)
  return table.concat({
    "name: " .. opts.name,
    "",
    "on:",
    "  push:",
    "  pull_request:",
    "",
    "jobs:",
    "  snap:",
    "    runs-on: ubuntu-latest",
    "    steps:",
    "      - uses: actions/checkout@v4",
    "      - name: Install Neovim",
    "        run: sudo apt-get update && sudo apt-get install -y neovim",
    "      - name: Run snapshots",
    "        run: |",
    "          nvim --headless -u NONE -i NONE -l snap.lua run --root " .. opts.root .. " --format json",
    "      - name: Compare snapshots",
    "        run: |",
    "          nvim --headless -u NONE -i NONE -l snap.lua compare --root " .. opts.root .. " --format "
      .. opts.format .. " --diff-always",
    "      - name: Upload diffs",
    "        if: always()",
    "        uses: actions/upload-artifact@v4",
    "        with:",
    "          name: nvim-snap-diff",
    "          path: |",
    "            **/diff/*",
    "",
  }, "\n")
end

local function write_file(path, contents, force)
  if vim.fn.filereadable(path) == 1 and not force then
    return nil, "file exists: " .. path
  end
  local dir = vim.fn.fnamemodify(path, ":h")
  if dir and dir ~= "." then
    vim.fn.mkdir(dir, "p")
  end
  local fd, err = io.open(path, "w")
  if not fd then
    return nil, err
  end
  fd:write(contents)
  fd:close()
  return true
end

function M.run(args_list)
  local args = vim.deepcopy(args_list or {})
  local sub = args[1]
  if sub == nil or vim.startswith(sub, "-") then
    util.err_write("ci command is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  table.remove(args, 1)

  if sub ~= "init" then
    util.err_write("unknown ci command: " .. sub)
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local opts = parse_args(args)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local path = vim.fs.normalize(opts.path)
  local ok, err = write_file(path, workflow_yaml(opts), opts.force)
  if not ok then
    util.err_write(err or "failed to write workflow")
    vim.cmd("cq")
    return
  end
  print(path)
end

return M

end)
__bundle_register("snap.command_compare", function(require, _LOADED, __bundle_register, __bundle_modules)
local normalize = require("snap.normalize")
local output = require("snap.output")
local png = require("snap.png")
local render = require("snap.render")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua core compare [options]",
    "",
    "options:",
    "  --actual PATH      Snapshot JSON path ('-' for stdin)",
    "  --expected PATH    Expected JSON path",
    "  --diff             Print unified diff on mismatch",
    "  --diff-format FMT  Diff source: text|ansi|html|png (default: text)",
    "  --diff-out PATH    Write diff to PATH (default: stdout)",
    "  -h, --help         Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    actual = nil,
    expected = nil,
    diff = false,
    diff_format = "text",
    diff_out = "-",
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--actual" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--actual requires a value")
      else
        opts.actual = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--actual=") then
      opts.actual = string.sub(arg, 10)
    elseif arg == "--expected" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--expected requires a value")
      else
        opts.expected = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--expected=") then
      opts.expected = string.sub(arg, 12)
    elseif arg == "--diff" then
      opts.diff = true
    elseif arg == "--diff-format" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--diff-format requires a value")
      else
        opts.diff_format = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--diff-format=") then
      opts.diff_format = string.sub(arg, 15)
    elseif arg == "--diff-out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--diff-out requires a value")
      else
        opts.diff_out = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--diff-out=") then
      opts.diff_out = string.sub(arg, 12)
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function read_input(path)
  if path == "-" then
    return io.read("*a")
  end
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

local function decode_json(text)
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    return nil, "failed to parse json: " .. tostring(decoded)
  end
  return decoded
end

local function deep_equal(a, b)
  if type(a) ~= type(b) then
    return false
  end
  if type(a) ~= "table" then
    return a == b
  end
  local seen = {}
  for key, value in pairs(a) do
    if not deep_equal(value, b[key]) then
      return false
    end
    seen[key] = true
  end
  for key, _ in pairs(b) do
    if not seen[key] then
      return false
    end
  end
  return true
end

local function escape_html(text)
  return (text:gsub("[&<>\"']", {
    ["&"] = "&amp;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ['"'] = "&quot;",
    ["'"] = "&#39;",
  }))
end

local function highlight_unified(diff_text)
  local out = { "<pre>" }
  for line in diff_text:gmatch("([^\n]*)\n?") do
    if line == "" and #out > 1 and out[#out] == "" then
      break
    end
    local cls = "line"
    if vim.startswith(line, "+") then
      cls = cls .. " add"
    elseif vim.startswith(line, "-") then
      cls = cls .. " del"
    elseif vim.startswith(line, "@@") then
      cls = cls .. " hunk"
    end
    table.insert(out, '<div class="' .. cls .. '">' .. escape_html(line) .. "</div>")
  end
  table.insert(out, "</pre>")
  return table.concat(out)
end

local function render_for_diff(snapshot, format)
  if format == "ansi" then
    return render.render_ansi(snapshot)
  end
  return render.render_text(snapshot)
end

local function render_html_diff(expected, actual, default_view)
  local expected_render_text = render.render_text(expected)
  local actual_render_text = render.render_text(actual)
  local unified = vim.text.diff(expected_render_text, actual_render_text, { result_type = "unified", ctxlen = 3 })
  local diff_map = build_diff_map(expected, actual)
  local expected_plain = render.render_html_cells(expected, diff_map.expected, "removed")
  local actual_plain = render.render_html_cells(actual, diff_map.actual, "added")
  local aligned = build_aligned_maps(expected, actual)
  local expected_aligned = render.render_html_aligned(
    expected,
    aligned.expected_rows,
    aligned.expected_line_kinds,
    aligned.expected_cells,
    "removed"
  )
  local actual_aligned = render.render_html_aligned(
    actual,
    aligned.actual_rows,
    aligned.actual_line_kinds,
    aligned.actual_cells,
    "added"
  )
  return wrap_html_diff(
    highlight_unified(unified or ""),
    expected_plain,
    actual_plain,
    expected_aligned,
    actual_aligned,
    default_view
  )
end

local function wrap_html_diff(unified_diff, expected_plain, actual_plain, expected_aligned, actual_aligned, default_view)
  if default_view ~= "side" and default_view ~= "overlay" then
    default_view = "unified"
  end
  local unified_checked = default_view == "unified" and " checked" or ""
  local side_checked = default_view == "side" and " checked" or ""
  local overlay_checked = default_view == "overlay" and " checked" or ""
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>nvim-snap compare</title>",
    "  <style>",
    "    :root { color-scheme: light; }",
    "    body { margin: 0; background: #f4f5f7; color: #1f2328; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }",
    "    .wrap { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; padding: 0; }",
    "    .panel { background: #ffffff; border: 1px solid #d0d7de; border-radius: 6px; overflow: auto; }",
    "    .title { padding: 8px 12px; border-bottom: 1px solid #222; font-weight: 600; }",
    "    .content { padding: 12px; }",
    "    .content pre { margin: 0; white-space: pre; }",
    "    .tabs { display: inline-flex; gap: 4px; padding: 10px 12px; margin: 12px 16px 0; align-items: center; background: #ffffff; border: 1px solid #d0d7de; border-radius: 10px; box-shadow: 0 1px 2px rgba(16, 24, 40, 0.08); }",
    "    .tabs .label { color: #57606a; font-size: 1em; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; padding: 4px 8px; border: 1px solid #e1e4e8; border-radius: 999px; background: #f6f8fa; }",
    "    .tabs label { color: #24292f; text-decoration: none; padding: 6px 12px; border: 1px solid transparent; border-radius: 8px; cursor: pointer; user-select: none; }",
    "    .tabs label:hover { background: #f6f8fa; }",
    "    .toggles { position: absolute; opacity: 0; pointer-events: none; }",
    "    #view-unified:checked ~ .tabs label[for=\"view-unified\"],",
    "    #view-side:checked ~ .tabs label[for=\"view-side\"],",
    "    #view-side-diff:checked ~ .tabs label[for=\"view-side-diff\"] {",
    "      color: #ffffff; border-color: #2da44e; background: #2da44e;",
    "    }",
    "    .section { padding: 12px 16px 16px; }",
    "    .cell.diff.added { background: rgba(34,197,94,0.28); box-shadow: inset 0 0 0 1px rgba(34,197,94,0.7); }",
    "    .cell.diff.removed { background: rgba(220,38,38,0.22); box-shadow: inset 0 0 0 1px rgba(220,38,38,0.7); }",
    "    .view-plain .line.diff, .view-plain .line.diff .cell, .view-plain .cell.diff { background: none; box-shadow: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.added .cell { box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .line.diff.removed .cell { box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.18); }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.added {",
    "      box-shadow: inset 0 0 0 9999px rgba(34,197,94,0.45), inset 0 0 0 1px rgba(34,197,94,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(34,197,94,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    #view-side-diff:checked ~ .page #side .view-aligned .cell.diff.removed {",
    "      box-shadow: inset 0 0 0 9999px rgba(220,38,38,0.4), inset 0 0 0 1px rgba(220,38,38,0.8);",
    "      text-decoration: underline;",
    "      text-decoration-color: rgba(220,38,38,0.9);",
    "      text-decoration-thickness: 2px;",
    "      text-underline-offset: 0.12em;",
    "    }",
    "    .udiff { color: #24292f; }",
    "    .udiff .line { padding: 0 4px; }",
    "    .udiff .line.add { background: rgba(34,197,94,0.18); }",
    "    .udiff .line.del { background: rgba(220,38,38,0.18); }",
    "    .udiff .line.hunk { color: #0969da; }",
    "    #view-unified:not(:checked) ~ .page #unified { display: none; }",
    "    #view-unified:checked ~ .page #side { display: none; }",
    "    #view-side:checked ~ .page #side { display: block; }",
    "    #view-side-diff:checked ~ .page #side { display: block; }",
    "    #view-side:checked ~ .page #side .view-plain { display: block; }",
    "    #view-side:checked ~ .page #side .view-aligned { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-plain { display: none; }",
    "    #view-side-diff:checked ~ .page #side .view-aligned { display: block; }",
    "    .grid { display: inline-block; }",
    "    .line { display: block; white-space: pre; }",
    "    .cell { display: inline-block; }",
    "  </style>",
    "</head>",
    "<body>",
    "  <input id=\"view-unified\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. unified_checked .. " />",
    "  <input id=\"view-side\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. side_checked .. " />",
    "  <input id=\"view-side-diff\" class=\"toggles\" type=\"radio\" name=\"diff-view\"" .. overlay_checked .. " />",
    "  <div class=\"tabs\">",
    "    <span class=\"label\">view</span>",
    "    <label for=\"view-unified\">unified</label>",
    "    <label for=\"view-side\">side</label>",
    "    <label for=\"view-side-diff\">overlay</label>",
    "  </div>",
    "  <div class=\"page\">",
    "  <div class=\"section\" id=\"unified\">",
    "    <div class=\"panel\">",
    "      <div class=\"title\">unified diff (text)</div>",
    "      <div class=\"content udiff\">" .. unified_diff .. "</div>",
    "    </div>",
    "  </div>",
    "  <div class=\"section\" id=\"side\">",
    "    <div class=\"wrap\">",
    "      <div class=\"panel\">",
    "        <div class=\"title\">expected</div>",
    "        <div class=\"content view-plain\" style=\"background:" .. expected_plain.bg .. ";color:" .. expected_plain.fg .. ";\"><div class=\"grid\">" .. expected_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. expected_aligned.bg .. ";color:" .. expected_aligned.fg .. ";\"><div class=\"grid\">" .. expected_aligned.html .. "</div></div>",
    "      </div>",
    "      <div class=\"panel\">",
    "        <div class=\"title\">actual</div>",
    "        <div class=\"content view-plain\" style=\"background:" .. actual_plain.bg .. ";color:" .. actual_plain.fg .. ";\"><div class=\"grid\">" .. actual_plain.html .. "</div></div>",
    "        <div class=\"content view-aligned\" style=\"background:" .. actual_aligned.bg .. ";color:" .. actual_aligned.fg .. ";\"><div class=\"grid\">" .. actual_aligned.html .. "</div></div>",
    "      </div>",
    "    </div>",
    "  </div>",
    "  </div>",
    "</body>",
    "</html>",
  }, "\n")
end

local function grid_text_matrix(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return 0, 0, {}
  end
  local rows = grid.rows or 0
  local cols = grid.cols or 0
  local matrix = {}
  for r = 1, rows do
    local row_cells = grid.cells[r] or {}
    local line = {}
    for c = 1, cols do
      local cell = row_cells[c] or { text = " " }
      local text = cell.text
      if text == "" then
        text = " "
      end
      line[c] = text
    end
    matrix[r] = line
  end
  return rows, cols, matrix
end

local function lines_from_matrix(rows, cols, matrix)
  local lines = {}
  for r = 1, rows do
    local row = matrix[r] or {}
    lines[r] = table.concat(row, "")
  end
  return lines
end

local function align_lines(expected_lines, actual_lines)
  local expected_text = table.concat(expected_lines, "\n")
  local actual_text = table.concat(actual_lines, "\n")
  local diffs = vim.text.diff(expected_text, actual_text, {
    result_type = "indices",
    algorithm = "patience",
    linematch = true,
    indent_heuristic = true,
  })
  local pairs = {}
  local e = 1
  local a = 1
  for _, d in ipairs(diffs) do
    local a_start, a_count, b_start, b_count = d[1], d[2], d[3], d[4]
    local expected_unchanged = (a_count == 0) and (a_start - e + 1) or (a_start - e)
    local actual_unchanged = (b_count == 0) and (b_start - a + 1) or (b_start - a)
    if expected_unchanged < 0 then
      expected_unchanged = 0
    end
    if actual_unchanged < 0 then
      actual_unchanged = 0
    end
    local common = math.min(expected_unchanged, actual_unchanged)
    for i = 0, common - 1 do
      table.insert(pairs, { e = e + i, a = a + i, kind = nil })
    end
    if expected_unchanged > common then
      for i = 0, expected_unchanged - common - 1 do
        table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
      end
    elseif actual_unchanged > common then
      for i = 0, actual_unchanged - common - 1 do
        table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
      end
    end
    local e_change = (a_count == 0) and (a_start + 1) or a_start
    local a_change = (b_count == 0) and (b_start + 1) or b_start
    e = e_change
    a = a_change
    local maxc = math.max(a_count, b_count)
    for i = 0, maxc - 1 do
      local er = (i < a_count) and (e + i) or 0
      local ar = (i < b_count) and (a + i) or 0
      local kind = nil
      if er == 0 and ar > 0 then
        kind = "added"
      elseif ar == 0 and er > 0 then
        kind = "removed"
      else
        kind = "changed"
      end
      table.insert(pairs, { e = er, a = ar, kind = kind })
    end
    e = e + a_count
    a = a + b_count
  end
  local expected_remaining = math.max(#expected_lines - e + 1, 0)
  local actual_remaining = math.max(#actual_lines - a + 1, 0)
  local common = math.min(expected_remaining, actual_remaining)
  for i = 0, common - 1 do
    table.insert(pairs, { e = e + i, a = a + i, kind = nil })
  end
  if expected_remaining > common then
    for i = 0, expected_remaining - common - 1 do
      table.insert(pairs, { e = e + common + i, a = 0, kind = "removed" })
    end
  elseif actual_remaining > common then
    for i = 0, actual_remaining - common - 1 do
      table.insert(pairs, { e = 0, a = a + common + i, kind = "added" })
    end
  end
  return pairs
end

local function build_aligned_maps(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local expected_lines = lines_from_matrix(erows, ecols, ematrix)
  local actual_lines = lines_from_matrix(arows, acols, amatrix)
  local pairs = align_lines(expected_lines, actual_lines)
  local cols = math.max(ecols, acols)
  local expected_rows = {}
  local actual_rows = {}
  local expected_line_kinds = {}
  local actual_line_kinds = {}
  local expected_cells = {}
  local actual_cells = {}
  for idx, pair in ipairs(pairs) do
    expected_rows[idx] = pair.e
    actual_rows[idx] = pair.a
    if pair.kind == "removed" then
      expected_line_kinds[idx] = "removed"
    elseif pair.kind == "added" then
      actual_line_kinds[idx] = "added"
    elseif pair.kind == "changed" then
      expected_line_kinds[idx] = "removed"
      actual_line_kinds[idx] = "added"
    end
    if pair.e and pair.e > 0 and pair.a and pair.a > 0 then
      for c = 1, cols do
        local etext = ematrix[pair.e] and ematrix[pair.e][c] or " "
        local atext = amatrix[pair.a] and amatrix[pair.a][c] or " "
        if etext ~= atext then
          expected_cells[pair.e] = expected_cells[pair.e] or {}
          actual_cells[pair.a] = actual_cells[pair.a] or {}
          expected_cells[pair.e][c] = true
          actual_cells[pair.a][c] = true
        end
      end
    end
  end
  return {
    expected_rows = expected_rows,
    actual_rows = actual_rows,
    expected_line_kinds = expected_line_kinds,
    actual_line_kinds = actual_line_kinds,
    expected_cells = expected_cells,
    actual_cells = actual_cells,
  }
end

local function build_diff_map(expected_snapshot, actual_snapshot)
  local erows, ecols, ematrix = grid_text_matrix(expected_snapshot)
  local arows, acols, amatrix = grid_text_matrix(actual_snapshot)
  local rows = math.max(erows, arows)
  local cols = math.max(ecols, acols)
  local expected = { lines = {}, cells = {} }
  local actual = { lines = {}, cells = {} }
  for r = 1, rows do
    local line_diff = false
    for c = 1, cols do
      local etext = ematrix[r] and ematrix[r][c] or " "
      local atext = amatrix[r] and amatrix[r][c] or " "
      if etext ~= atext then
        expected.cells[r] = expected.cells[r] or {}
        actual.cells[r] = actual.cells[r] or {}
        expected.cells[r][c] = true
        actual.cells[r][c] = true
        line_diff = true
      end
    end
    if line_diff then
      expected.lines[r] = true
      actual.lines[r] = true
    end
  end
  return { expected = expected, actual = actual }
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if not opts.actual or opts.actual == "" then
    util.err_write("--actual is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if not opts.expected or opts.expected == "" then
    util.err_write("--expected is required")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if
    opts.diff_format ~= "text"
    and opts.diff_format ~= "ansi"
    and opts.diff_format ~= "html"
    and opts.diff_format ~= "png"
  then
    util.err_write("--diff-format must be text, ansi, html, or png")
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local actual_text, actual_err = read_input(opts.actual)
  if not actual_text then
    util.err_write(actual_err or "failed to read actual")
    vim.cmd("cq")
    return
  end
  local actual_snapshot, actual_decode_err = decode_json(actual_text)
  if not actual_snapshot then
    util.err_write(actual_decode_err or "failed to parse actual")
    vim.cmd("cq")
    return
  end

  local expected_text, expected_err = read_input(opts.expected)
  if not expected_text then
    util.err_write(expected_err or "failed to read expected")
    vim.cmd("cq")
    return
  end
  local expected_snapshot, decode_err = decode_json(expected_text)
  if not expected_snapshot then
    util.err_write(decode_err or "failed to parse expected")
    vim.cmd("cq")
    return
  end

  local normalized_actual = normalize.normalize(actual_snapshot)
  local normalized_expected = normalize.normalize(expected_snapshot)

  if deep_equal(normalized_actual, normalized_expected) then
    print("match")
    return
  end
  if opts.diff then
    if opts.diff_format == "html" or opts.diff_format == "png" then
      if opts.diff_format == "png" and opts.diff_out == "-" then
        util.err_write("--diff-out must be a file path for png")
        vim.cmd("cq")
        return
      end
      local default_view = opts.diff_format == "png" and "overlay" or "unified"
      local rendered = render_html_diff(normalized_expected, normalized_actual, default_view)
      if opts.diff_format == "html" then
        local ok, write_err = output.write(opts.diff_out, rendered)
        if not ok then
          util.err_write(write_err or "failed to write diff")
          vim.cmd("cq")
          return
        end
      else
        local ok, write_err = png.write_png_from_html(rendered, opts.diff_out)
        if not ok then
          util.err_write(write_err or "failed to write diff png")
          vim.cmd("cq")
          return
        end
      end
    else
      local expected_out = render_for_diff(normalized_expected, opts.diff_format)
      local actual_out = render_for_diff(normalized_actual, opts.diff_format)
      local diff = vim.text.diff(expected_out, actual_out, { result_type = "unified", ctxlen = 3 })
      if diff then
        local ok, write_err = output.write(opts.diff_out, diff)
        if not ok then
          util.err_write(write_err or "failed to write diff")
          vim.cmd("cq")
          return
        end
      end
    end
  end

  util.err_write("mismatch")
  vim.cmd("cq")
end

return M

end)
__bundle_register("snap.command_normalize", function(require, _LOADED, __bundle_register, __bundle_modules)
local normalize = require("snap.normalize")
local output = require("snap.output")
local util = require("snap.util")

local M = {}

local function usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua core normalize [options]",
    "",
    "options:",
    "  --in PATH       Input JSON path ('-' for stdin)",
    "  --out PATH      Output JSON path ('-' for stdout)",
    "  --pretty        Pretty-print JSON",
    "  -h, --help      Show this help",
  }, "\n")
end

local function parse_args(args)
  local opts = {
    input = "-",
    output = "-",
    pretty = false,
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--in" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--in requires a value")
      else
        opts.input = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--in=") then
      opts.input = string.sub(arg, 6)
    elseif arg == "--out" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--out requires a value")
      else
        opts.output = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--out=") then
      opts.output = string.sub(arg, 7)
    elseif arg == "--pretty" then
      opts.pretty = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  return opts
end

local function read_input(path)
  if path == "-" then
    return io.read("*a")
  end
  local fd, err = io.open(path, "r")
  if not fd then
    return nil, err
  end
  local text = fd:read("*a")
  fd:close()
  return text
end

function M.run(args_list)
  local opts = parse_args(args_list)
  if opts.help then
    print(usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(usage())
    vim.cmd("cq")
    return
  end

  local text, err = read_input(opts.input)
  if not text then
    util.err_write(err or "failed to read input")
    vim.cmd("cq")
    return
  end
  local ok, decoded = pcall(vim.json.decode, text)
  if not ok then
    util.err_write("failed to parse json: " .. tostring(decoded))
    vim.cmd("cq")
    return
  end

  local normalized = normalize.normalize(decoded)
  local encoded = vim.json.encode(normalized)
  if opts.pretty then
    encoded = vim.json.encode(normalized, { indent = "  " })
  end

  local ok_write, write_err = output.write(opts.output, encoded)
  if not ok_write then
    util.err_write(write_err or "failed to write output")
    vim.cmd("cq")
  end
end

return M

end)
__bundle_register("snap.command_capture", function(require, _LOADED, __bundle_register, __bundle_modules)
local args = require("snap.args")
local output = require("snap.output")
local render = require("snap.render")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

local function normalize_out_dir(path)
  if not path then
    return nil
  end
  return vim.fs.normalize(vim.fn.fnamemodify(path, ":p"))
end

function M.run(args_list)
  local opts = args.parse(args_list)
  if opts.help then
    print(args.usage())
    return
  end
  if opts.invalid then
    util.err_write("invalid args:")
    for _, msg in ipairs(opts.invalid) do
      util.err_write("  " .. msg)
    end
    util.err_write(args.usage())
    vim.cmd("cq")
    return
  end
  if opts.unknown then
    util.err_write("unknown args: " .. table.concat(opts.unknown, " "))
    util.err_write(args.usage())
    vim.cmd("cq")
    return
  end
  if #opts.scenarios == 0 then
    util.err_write("--scenario is required")
    util.err_write(args.usage())
    vim.cmd("cq")
    return
  end
  if not opts.out_dir or opts.out_dir == "" then
    util.err_write("--out-dir is required")
    util.err_write(args.usage())
    vim.cmd("cq")
    return
  end

  opts.scripts = opts.scenarios
  local out_dir = normalize_out_dir(opts.out_dir)
  if not out_dir then
    util.err_write("failed to resolve --out-dir")
    vim.cmd("cq")
    return
  end

  local result, err = snapshot.collect(opts)
  if not result then
    util.err_write(err or "snapshot failed")
    vim.cmd("cq")
    return
  end

  if opts.json then
    local encoded = vim.json.encode(result)
    local ok, write_err = output.write(vim.fs.joinpath(out_dir, "snapshot.json"), encoded)
    if not ok then
      util.err_write(write_err or "failed to write json output")
      vim.cmd("cq")
    end
  end
  if opts.ansi then
    local ansi = render.render_ansi(result)
    local ok_ansi, err_ansi = output.write(vim.fs.joinpath(out_dir, "snapshot.ansi"), ansi)
    if not ok_ansi then
      util.err_write(err_ansi or "failed to write ansi output")
      vim.cmd("cq")
    end
  end
  if opts.html then
    local html = render.render_html(result)
    local ok_html, err_html = output.write(vim.fs.joinpath(out_dir, "snapshot.html"), html)
    if not ok_html then
      util.err_write(err_html or "failed to write html output")
      vim.cmd("cq")
    end
  end
end

return M

end)
__bundle_register("snap.args", function(require, _LOADED, __bundle_register, __bundle_modules)
local M = {}

function M.usage()
  return table.concat({
    "usage:",
    "  nvim -l snap.lua core capture [options]",
    "",
    "options:",
    "  --scenario PATH    Lua scenario file to run (repeatable)",
    "  --out-dir PATH     Output directory for snapshot files",
    "  --json             Write snapshot.json (default)",
    "  --ansi             Write snapshot.ansi",
    "  --html             Write snapshot.html",
    "  --width N          UI columns (default: 80)",
    "  --height N         UI lines (default: 24)",
    "  --data-home PATH   XDG_DATA_HOME for embedded nvim",
    "  --config-home PATH XDG_CONFIG_HOME for embedded nvim",
    "  --wait MS          Wait for redraw flush (default: 200)",
    "  --rpc-timeout MS   RPC request timeout (default: 2000)",
    "  --nvim PATH        Embedded nvim path (default: nvim)",
    "  --multigrid        Enable ext_multigrid",
    "  -h, --help         Show this help",
  }, "\n")
end

function M.parse(args)
  local opts = {
    scenarios = {},
    out_dir = nil,
    json = false,
    ansi = false,
    html = false,
    wait = 200,
    width = 80,
    height = 24,
    nvim = "nvim",
    rpc_timeout = 2000,
    data_home = nil,
    config_home = nil,
    multigrid = false,
    _flags = {
      output = false,
    },
  }
  local i = 1
  while i <= #args do
    local arg = args[i]
    if arg == "--scenario" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--scenario requires a value")
      else
        table.insert(opts.scenarios, value)
        i = i + 1
      end
    elseif vim.startswith(arg, "--scenario=") then
      table.insert(opts.scenarios, string.sub(arg, 12))
    elseif arg == "--out-dir" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--out-dir requires a value")
      else
        opts.out_dir = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--out-dir=") then
      opts.out_dir = string.sub(arg, 11)
    elseif arg == "--json" then
      opts.json = true
      opts._flags.output = true
    elseif arg == "--ansi" then
      opts.ansi = true
      opts._flags.output = true
    elseif arg == "--html" then
      opts.html = true
      opts._flags.output = true
    elseif arg == "--width" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--width requires a value")
      else
        opts.width = tonumber(value) or opts.width
        i = i + 1
      end
    elseif vim.startswith(arg, "--width=") then
      opts.width = tonumber(string.sub(arg, 9)) or opts.width
    elseif arg == "--height" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--height requires a value")
      else
        opts.height = tonumber(value) or opts.height
        i = i + 1
      end
    elseif vim.startswith(arg, "--height=") then
      opts.height = tonumber(string.sub(arg, 10)) or opts.height
    elseif arg == "--data-home" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--data-home requires a value")
      else
        opts.data_home = vim.fn.fnamemodify(value, ":p")
        i = i + 1
      end
    elseif vim.startswith(arg, "--data-home=") then
      opts.data_home = vim.fn.fnamemodify(string.sub(arg, 12), ":p")
    elseif arg == "--config-home" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--config-home requires a value")
      else
        opts.config_home = vim.fn.fnamemodify(value, ":p")
        i = i + 1
      end
    elseif vim.startswith(arg, "--config-home=") then
      opts.config_home = vim.fn.fnamemodify(string.sub(arg, 14), ":p")
    elseif arg == "--wait" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--wait requires a value")
      else
        opts.wait = tonumber(value) or opts.wait
        i = i + 1
      end
    elseif vim.startswith(arg, "--wait=") then
      opts.wait = tonumber(string.sub(arg, 8)) or opts.wait
    elseif arg == "--rpc-timeout" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--rpc-timeout requires a value")
      else
        opts.rpc_timeout = tonumber(value) or opts.rpc_timeout
        i = i + 1
      end
    elseif vim.startswith(arg, "--rpc-timeout=") then
      opts.rpc_timeout = tonumber(string.sub(arg, 15)) or opts.rpc_timeout
    elseif arg == "--nvim" then
      local value = args[i + 1]
      if value == nil then
        opts.invalid = opts.invalid or {}
        table.insert(opts.invalid, "--nvim requires a value")
      else
        opts.nvim = value
        i = i + 1
      end
    elseif vim.startswith(arg, "--nvim=") then
      opts.nvim = string.sub(arg, 8)
    elseif arg == "--multigrid" then
      opts.multigrid = true
    elseif arg == "--help" or arg == "-h" then
      opts.help = true
    else
      opts.unknown = opts.unknown or {}
      table.insert(opts.unknown, arg)
    end
    i = i + 1
  end
  if not opts._flags.output then
    opts.json = true
  end
  return opts
end

return M

end)
return __bundle_require("__root")