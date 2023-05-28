---
marp: true
style: |
  .columns {
    display: flex;
    gap: 1rem;
  }

  .grid2 {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }
---

<!--
theme: default
class: invert
paginate: true
-->

# Suffiks og Stout

To og en haug eksperimenter

<!--
 Suffiks var en test påbegynt i slutten av 2021 for å teste en idé etter samtaler med Skatt om å gjennbruke Naiserator.

 Stout ble startet i 2022 for å teste et par idéer om hvordan gjøre utvikleropplevelsen nærmere det som leveres av Fly.io, Heroku og lignende.
-->

---

# Hva er Suffiks

Suffiks er en Kubernetes Operator som gjør det enkelt å kjøre applikasjoner i Kubernetes.

Sterkt inspirert av Naiserator, men med støtte for utvidelser.

<!--
Inspirert av samtalene rundt at Skatt tenkte å ta ibruk Naiserator. Tanken var å gjøre det enkelt å kjøre applikasjoner i Kubernetes, og å gjøre det enkelt å tilpasse til ulike behov.

Selve spec-en som leveres av Suffiks er begrenset til det absolutte minimumet.
F.eks. er ikke replicas en del av spec-en, da dette kan variere etter behov (f.eks. ved å støtte Horizontal Pod Autoscaler).
-->

---

# Uvidelser

Utvidelser er en måte å utvide spec-en på.

Disse er implementert som en GRPC server for å ikke begrense seg til ett spesifikt språk.

Utvidelser innholder også egen dokumentasjon.

<!--
Siden tanken med Suffiks er å ha en kjærne som man kan utvide, så skal det være lett for team som ikke kjenner Go å kunne utvide Suffiks.

Dokumentasjon for det utvidelsen leverer er en del av extension-en, og det kan derfor genereres dokumentasjonssider som er spesifikk for en installasjon.
-->

---

<div class="columns" style="align-items: end">
<div>

```yaml
apiVersion: suffiks.com/v1
kind: Extension
metadata:
  name: traefik
spec:
  controller:
    namespace: extensions
    port: 8383
    service: suffiks-traefik
  targets: [Application]
  webhooks:
    validation: true
  openAPIV3Schema:
    properties:
      ingresses:
        items:
          properties:
            host: { type: string }
            path: { type: string }
          required: [host]
          type: object
        type: array
    type: object
```

</div>

<div>

- Definer en Extensionen `traefik`
- Den kjører på `suffiks-traefik.extensions:8383`
- Utvid `Application` med `ingresses`

Vi støtter med det å gjøre følgende i `Application`:

```yaml
# ...
spec:
  # ...
  ingresses:
    - host: hello.suffiks.com
      path: /
```

</div>
</div>

<!--
Lager en extension som støtter å legge til en ingress i Application-en.

Controller definerer hvor extension-en kjører, og hvilken service som skal brukes for å nå den.

Targets definerer hvilke typer objekter extension-en utvider.

Webhooks definerer hvilke webhooks som extension-en støtter.
-->

---

# Demo

<!--
I et cluster med Suffiks installert (For å teste å deploye en app før installasjon av traefik, se nederst i kommentaren):

Sjekk også ut stout.suffiks.com for å se på dokumentasjonen.

For å se gjeldende properties på Application.spec:
$ kubectl get crd applications.suffiks.com -o json | jq '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties | keys'

Installert traefik:
$ helm install --namespace extensions --create-namespace --version 0.1.3 suffiks-traefik oci://ghcr.io/suffiks/charts/extension-traefik

$ kubectl get ext

Se endringer i Application.spec:
$ kubectl get crd applications.suffiks.com -o json | jq '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties | keys'

Og:

$ kubectl get crd applications.suffiks.com -o json | jq '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.ingresses.items.properties | keys'

apiVersion: suffiks.com/v1
kind: Application
metadata:
  name: demo-app
spec:
  image: localhost:5000/test-team2/stout-example-app:build-1675504033
  ingresses:
    - host: hello.suffiks.com
      path: /

Dersom opentelemetry er konfigurert med tracing, ta en titt på tracingen.


---

Lage ny extension:

$ extgen new --target Application --validation --defaulting github.com/suffiks/extensions/demo
$ cd demo
$ go mod tidy
$ go test ./...


-->

---

# Hva er Stout

Stout er et eksperiment for å tilby en annen utvikleropplevelse ved å abstrahere bort nødvendig kunnskap om Kubernetes.

<!--
Tanker som jeg har vært innom:

- Alt av autentisering og autorisering for å deploye og vedlikeholde apper gjøres gjennom ett system.
- Fjerne behovet for YAML og å forholde seg til Kubernetes API-et.
- Fjerne behovet for å lage Dockerfiles.
- Oversikt og kontroll over applikasjoner fra et web-grensesnitt.
- Gjøre mest mulig med OpenID Connect (Workload Identity).
-->

---

# En ny deploy

<div class="grid2">
<div>
Fremfor å spesifisere hver bit av en Application-YAML, så vil mange apper kunne deployes med litt TOML:
</div>

<div>

```toml
# stout.toml
name = "stout-example-app"
team = "test-team2"
```

</div>

</div>

<!--
Denne TOML filen inneholder det minimale for å deploye en app, navnet på appen og teamet det tilhører.

Stout-cli vil bruke buildpacks for å bygge et Docker image, dytte imaget til Stout registry, og API-et vil generere nødvendig YAML basert på TOML fila for å deploye appen.
-->

---

# ~~Procrastination~~ Frustrasjon 😢

Stout var det eksperimentet som ballet på seg og ble det mest frustrerende.

Mange idéer er tenkt og glemt rundt hvilke muligheter som ligger her.

Kanskje det som er mest interessant for NAIS, er å se på muligheten for å bruke buildpacks og generere YAML.

Kanskje også å se på om vi kan benytte noe av teknologien til [Ory](https://www.ory.sh/).

<!--
Stout som tjeneste endte opp med å balle på seg med veldig mange byggeklosser som måtte på plass før det kunne bli noe.

Med litt liten progregresjon, så ble det mye frustrasjon.

docker-compose fila inneholder 10 services, hvor 7 var tenkt nødvendige. Dette er uten å ta med kubernetes og Suffiks.

Ory tilbyr en del kule tjenester som Kratos for å håndtere brukere, og Keto for å håndtere tilgangskontroll.

Det er en del idéer i Stout som kan være med på å gi en potensielt bedre utvikleropplevelse:
- Buildpacks
- Generering av YAML
- Web app for å håndtere applikasjoner

Og andre ting inspirert av fly.io, Heroku, render.com, Vercel osv.

-->

---

# Konklusjon om Suffiks?

Suffiks tror jeg kunne gitt verdi for et par bruksområder vi har i dag:

- La tenants utvide applikasjonsspec-en for egne behov.
- Generere dokumentasjon for de bitene av applikasjonsspec-en som er tilgjengelig per tenant.
- Separere ut logiske biter av Naiserator i egne extensions.
- Utnytte webhooks mer

<!--
Idéen om å ha en operator som kan utvides virker å være mulig. Dette kan være en interessant måte å la tenants og andre utnytte plattformen men samtidig kunne tilpasse den.
-->

---

# Uvissheter om Suffiks

- Hva er kjerne-spec, hva er extensions?
- Suffiks har ikke kjørt noe særlig utenom på min maskin, med et relativt få apper, så er usikkerhet på hvordan det vil fungere i større skala.

<!--
Hva vil være kjerne-spec-en til Suffiks? Hva vil være extensions?

Hvordan vil Suffiks skalere med flere extensions og mange apper?

-->

---

# Konklusjon om Stout?

Kan ikke si jeg har mange konklusjoner her.
Tror det er noen konsepter som hadde vært interessant å se på nærmere, som buildpacks en mindre TOML-spec.

---

# Takk for meg 😗
