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
