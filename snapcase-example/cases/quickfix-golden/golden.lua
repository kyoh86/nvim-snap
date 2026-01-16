vim.cmd("enew")
vim.fn.setline(1, {
  "Quickfix Preview",
  "search: TODO",
  "",
  "Use :cnext / :cprev to jump",
})

vim.fn.setqflist({
  { filename = "lua/foo.lua", lnum = 12, col = 3, text = "undefined global" },
  { filename = "lua/bar.lua", lnum = 4, col = 1, text = "unused local" },
  { filename = "README.md", lnum = 2, col = 1, text = "typo" },
}, "r")

vim.cmd("copen")
vim.cmd("wincmd p")
vim.cmd("setlocal nonumber")
vim.cmd("setlocal norelativenumber")
