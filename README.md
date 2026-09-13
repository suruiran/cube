# cube

A collection of Go packages used in my personal projects, [DBNova](https://dbnova.ruiransoft.com).

# pkgs

- action

  A http framework. Expose actions in `init` functions, then use `cube/x/tools/autoimport` to generate an import file.
  It also implements host-admin auth for dockerized services based on readonly fs.

- dic

  A very simple yet powerful di container.

- udshttp

  An HTTP implementation based on unix domain socket. It keep the `ctx.Cancel`'s ability between processes.
  DBNova use this to communicate with its external driver.

- rbc

  Init-time reflection framework. The `rbc` stands for `Rubick`, a `Dota 2` hero. (I do not use this pkg in real projects yet.)

- sqlx

  A light extension for `database/sql` based on `rbc`.

The rest are utility packages.