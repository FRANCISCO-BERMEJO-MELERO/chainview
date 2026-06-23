# Contribuir a chainview

¡Gracias por tu interés! chainview es un monitor de wallets EVM en el terminal,
watch-only y keyless. Esta guía resume cómo levantar el entorno, el estilo del
código y cómo proponer cambios.

## Requisitos

- **Go 1.25** o superior.
- `make` (opcional pero recomendado; los targets envuelven los comandos `go`).
- [`golangci-lint`](https://golangci-lint.run) **v2** para pasar el linter en
  local (la CI usa la v2.12.2).

## Poner en marcha

```sh
make setup    # go mod download + tidy
make run      # arranca la TUI sin compilar a disco
make build    # binario en ./bin/chainview con la versión inyectada
```

`chainview --help` lista las opciones; `chainview --version` muestra la versión
de build. Para diagnosticar, `chainview --debug` (o `CHAINVIEW_DEBUG=1`).

## Antes de abrir un PR

Deja el árbol en verde con lo mismo que corre la CI:

```sh
gofmt -l .            # no debe listar nada
go vet ./...
make test             # go test -race ./...
make lint             # golangci-lint run
```

Los **tests golden** de la TUI comparan frames renderizados. Si cambias el
render a propósito, regenera las snapshots y revisa el diff antes de
commitearlo:

```sh
go test ./internal/ui/ -run TestGolden -update
```

## Estilo

- Código formateado con `gofmt`; el linter (`.golangci.yml`) es la referencia.
- **Comentarios y mensajes de usuario en español**, como el resto del repo.
  Explican el *por qué*, no el *qué*.
- Errores en minúscula y envueltos con contexto (`fmt.Errorf("...: %w", err)`).
- Sin API keys ni dependencias que rompan el arranque keyless por defecto.

## Commits y ramas

- Trabaja en una rama de feature; los PR van contra `main`.
- Mensajes de commit **naturales y descriptivos en español**, en imperativo
  («Añade…», «Corrige…»). Sin prefijos de tipo, sin IDs de tarea y sin líneas
  de co-autoría. Un commit por cambio lógico.

## Diseño y decisiones

Las features grandes se diseñan antes de implementarlas. Las specs viven en
`docs/` (carpeta local, no versionada): describen objetivo, decisiones y orden
de implementación. Si propones algo de calado, abre antes una issue para
acordar el enfoque.

## Reportar bugs y pedir features

Usa las plantillas de issue. Para bugs, incluye versión (`chainview --version`),
SO/terminal y pasos para reproducir. Para vulnerabilidades, contacta en privado
en lugar de abrir una issue pública.

## Licencia

Al contribuir aceptas que tu aportación se publique bajo la licencia
[MIT](./LICENSE) del proyecto.
