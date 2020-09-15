# ShareItGo

Just another simple serving static command line tool.

Sometime we need to share something on HTTP briefly, so I write a simple program to do it.

## Installation

```
go install github.com/kmollee/shareitgo
```

## Usage

```
Usage of shareitgo:
  -d string
      the directory of static file to host (default ".")
  -ip string
      host addr
  -p int
      port to listen (default 3000)
```

## Example

1. share current on address `:3000`

   ```
   shareitgo
   ```

2. share video directory on specific address `192.168.82.28:10080`

   ```
   shareitgo -ip 192.168.82.28 -p 10080 -d ~/Videos
   ```
