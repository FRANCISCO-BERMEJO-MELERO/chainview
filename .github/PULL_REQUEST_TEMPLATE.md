<!-- Gracias por contribuir. Rellena lo relevante y borra lo que no aplique. -->

## Qué cambia

<!-- Resumen del cambio y motivación. Enlaza la issue si existe (Closes #N). -->

## Tipo de cambio

- [ ] Bug fix
- [ ] Nueva feature
- [ ] Refactor / interno
- [ ] Documentación

## Comprobaciones

- [ ] `gofmt -l .` no lista nada
- [ ] `go vet ./...` pasa
- [ ] `make test` (`go test -race ./...`) en verde
- [ ] `make lint` (`golangci-lint run`) sin issues
- [ ] Si cambia el render de la TUI, regeneré las snapshots golden (`-update`) y revisé el diff

## Notas

<!-- Capturas, decisiones de diseño, cosas pendientes... -->
