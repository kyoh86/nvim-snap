local args = require("snap.args")
local case = require("snap.case")
local output = require("snap.output")
local render = require("snap.render")
local snapshot = require("snap.snapshot")
local util = require("snap.util")

local M = {}

function M.run(args_list)
  local opts = args.parse(args_list)
  if opts.help then
    print(args.usage())
    return
  end
  if opts.case then
    if opts._flags.json_out or opts._flags.ansi_out or opts._flags.html_out or opts._flags.script then
      util.err_write("--case cannot be combined with output/script options")
      util.err_write(args.usage())
      vim.cmd("cq")
      return
    end
    local loaded, err = case.load(opts)
    if not loaded then
      util.err_write(err or "failed to load case")
      vim.cmd("cq")
      return
    end
    opts = loaded
  end
  if opts.json_out == "none" then
    opts.json_out = nil
  end
  local stdout_count = 0
  for _, out in ipairs({ opts.json_out, opts.ansi_out, opts.html_out }) do
    if out == "-" then
      stdout_count = stdout_count + 1
    end
  end
  if stdout_count > 1 then
    util.err_write("stdout is shared by multiple outputs; choose one")
    util.err_write(args.usage())
    vim.cmd("cq")
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

  local result, err = snapshot.collect(opts)
  if not result then
    util.err_write(err or "snapshot failed")
    vim.cmd("cq")
    return
  end

  if opts.json_out then
    local encoded = vim.json.encode(result)
    local ok, write_err = output.write(opts.json_out, encoded)
    if not ok then
      util.err_write(write_err or "failed to write output")
      vim.cmd("cq")
    end
  end
  if opts.ansi_out then
    local ansi = render.render_ansi(result)
    local ok_ansi, err_ansi = output.write(opts.ansi_out, ansi)
    if not ok_ansi then
      util.err_write(err_ansi or "failed to write ansi output")
      vim.cmd("cq")
    end
  end
  if opts.html_out then
    local html = render.render_html(result)
    local ok_html, err_html = output.write(opts.html_out, html)
    if not ok_html then
      util.err_write(err_html or "failed to write html output")
      vim.cmd("cq")
    end
  end
end

return M
