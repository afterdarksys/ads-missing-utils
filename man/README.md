# Missing Utils manual pages

This directory contains source roff manual pages for every executable in
`cmd/`. Pages use section 1 (user commands) and are intentionally kept next to
the project source so reviews can verify CLI and documentation changes together.

Build compressed installable pages with:

```sh
make man-build
./build.sh man-build
```

Install them below a chosen prefix with either `make man-install PREFIX=/path`
or `./build.sh man-install prefix=/path`. The default prefix is
`/usr/local`; the destination is `share/man/man1`.

The pages describe the current CLI and are not a substitute for a security
boundary. Commands marked as MVPs or partial in the project documentation say
so in their individual page. Exit status follows the shared CLI convention:
0 success, 1 a completed negative finding or policy failure, 2 invalid usage,
3 partial visibility or partial result, and 4 a runtime failure. A command may
use only the statuses relevant to its contract.

## Commands

- [aaas.1](aaas.1) — `aaas`
- [accesswhy.1](accesswhy.1) — `accesswhy`
- [amtm.1](amtm.1) — `amtm`
- [authwhy.1](authwhy.1) — `authwhy`
- [binarywhy.1](binarywhy.1) — `binarywhy`
- [binparse.1](binparse.1) — `binparse`
- [bundleinfo.1](bundleinfo.1) — `bundleinfo`
- [cded.1](cded.1) — `cded`
- [certwhy.1](certwhy.1) — `certwhy`
- [clt.1](clt.1) — `clt`
- [driftwhy.1](driftwhy.1) — `driftwhy`
- [dtp.1](dtp.1) — `dtp`
- [envsub.1](envsub.1) — `envsub`
- [expose.1](expose.1) — `expose`
- [hashsum.1](hashsum.1) — `hashsum`
- [homestate.1](homestate.1) — `homestate`
- [incidentsnap.1](incidentsnap.1) — `incidentsnap`
- [info2logic.1](info2logic.1) — `info2logic`
- [jsondiff.1](jsondiff.1) — `jsondiff`
- [jsongate.1](jsongate.1) — `jsongate`
- [jsonprobe.1](jsonprobe.1) — `jsonprobe`
- [jwalk.1](jwalk.1) — `jwalk`
- [logic.1](logic.1) — `logic`
- [man2logic.1](man2logic.1) — `man2logic`
- [meow.1](meow.1) — `meow`
- [metascore.1](metascore.1) — `metascore`
- [netwhy.1](netwhy.1) — `netwhy`
- [patchwhy.1](patchwhy.1) — `patchwhy`
- [pcapwhy.1](pcapwhy.1) — `pcapwhy`
- [pll.1](pll.1) — `pll`
- [ports.1](ports.1) — `ports`
- [portwhy.1](portwhy.1) — `portwhy`
- [pwatch.1](pwatch.1) — `pwatch`
- [regocheck.1](regocheck.1) — `regocheck`
- [sandboxdiff.1](sandboxdiff.1) — `sandboxdiff`
- [servicewhy.1](servicewhy.1) — `servicewhy`
- [selinuxwhy.1](selinuxwhy.1) — `selinuxwhy`
- [spacelift-helper.1](spacelift-helper.1) — `spacelift-helper`
- [tfchanges.1](tfchanges.1) — `tfchanges`
- [tjt.1](tjt.1) — `tjt`
- [restartwhy.1](restartwhy.1) — `restartwhy`
- [unitwhy.1](unitwhy.1) — `unitwhy`
- [varmerge.1](varmerge.1) — `varmerge`
- [webkit-tool.1](webkit-tool.1) — `webkit-tool`
