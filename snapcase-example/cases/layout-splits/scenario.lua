vim.cmd("enew")
vim.cmd("set number")
vim.cmd("set relativenumber")
vim.cmd("set cursorline")
vim.fn.setline(1, {
  "Tasks",
  "- review design notes",
  "- update snapshot cases",
  "- verify layout output",
  "",
  "Notes",
  "  * keep paths stable",
  "  * show multi-window",
})
vim.api.nvim_win_set_cursor(0, { 2, 0 })

vim.cmd("vsplit")
vim.cmd("wincmd l")
vim.cmd("enew")
vim.cmd("setlocal nonumber")
vim.cmd("setlocal norelativenumber")
vim.fn.setline(1, {
  "Log",
  "10:14 boot",
  "10:15 sync",
  "10:18 render",
  "10:20 done",
})
vim.api.nvim_win_set_cursor(0, { 4, 0 })

vim.cmd("split")
vim.cmd("wincmd j")
vim.cmd("enew")
vim.cmd("setlocal nonumber")
vim.cmd("setlocal norelativenumber")
vim.fn.setline(1, {
  "Summary",
  "OK: 3",
  "WARN: 1",
  "FAIL: 0",
})
