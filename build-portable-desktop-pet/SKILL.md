---
name: build-portable-desktop-pet
description: Use when turning one or more visual references into a portable single-file Windows 10/11 x64 desktop pet, especially when character identity, asymmetric markings, animation states, click or drag interactions, transparent Win32 rendering, or EXE packaging must remain consistent.
---

# Build Portable Desktop Pet

## Contract

Turn readable visual references directly into a compact, character-faithful Windows 10/11 x64 desktop pet. Ask zero questions before starting unless an input file is unreadable. Default to silent, offline, no-install, no-admin, and no-autostart operation.

## Compact Workflow

Follow this exact sequence:

`reference inspection -> identity-lock.md -> one 8x11 atlas + icon -> scaffold_pet.py -> tests/build -> verify-delivery.sh -> EXE + usage + SHA256`

## Identity Lock

Inspect every reference. Treat `identity-lock.md` as a required visual deliverable and write it before generating raster assets. Record silhouette, palette, front/back features, asymmetric markings, and character-relative left/right directions. Treat named markings as invariants and prohibit mirroring that moves them. Do not invent lore, body parts, markings, clothes, or accessories unsupported by the references or request.

## Visual Assets

**REQUIRED SUB-SKILL:** Use `imagegen` for raster generation.

Generate one transparent 8-column × 11-row atlas matching the bundled runtime contract, including independent directional movement and directional-look rows. Cover the default runtime states: Idle, Walk, Run, Sleep, Think, Happy, Drag. Generate one app icon. Check identity, atlas dimensions and cells, alpha edges, and directional markings.

Keep the runtime reuse explicit: Run reuses the directional Walk rows at a faster rate; Sleep reuses Idle at a slower rate; Drag holds the first Idle frame.

Do not mandate a six-view Character Sheet, an eight-expression board, multi-round blind review, or a separate atlas per action.

## Runtime and Build

Run:

```bash
scripts/scaffold_pet.py --name NAME --atlas ATLAS --icon ICON --output-dir PROJECT
```

Use the scaffolded transparent, borderless, always-on-top Win32 GUI runtime. Bind interactions exactly: single click -> Happy; double click -> Sleep/Wake; left-drag -> Drag/move; right click -> menu. Run its tests, then its Windows x64 build script.

Build on a Darwin or Linux build host with Python 3 with Pillow; the toolchain fetcher rejects other hosts clearly.

## Verification

Run `scripts/verify-delivery.sh EXE ABSOLUTE_PROJECT_ROOT`. Require a Windows x64 GUI PE (`PE32+`), `.rsrc`, allowed system DLL imports, no leaked absolute project path, and printed SHA-256. Keep Windows 10/11 real-machine interaction verification explicitly separate from cross-build/static verification.

## Escalation

**REQUIRED SUB-SKILL:** Use `hatch-pet` only when identity drift, direction ambiguity, transparency defects, formal Character Sheet output, or repair loops make this compact path insufficient.

## Delivery

Return `identity-lock.md`, atlas, icon, scaffolded source, portable EXE, concise usage text, and SHA-256.
