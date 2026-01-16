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
