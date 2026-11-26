# Slop Town system requirements

**Phase 0 status, 25 November 2026.** Slop Town currently has a terminal program that saves and reloads a small test fixture. A playable release and its tested minimum requirements are still ahead. The intended desktop experience is to install the game, open it, and play offline with saves on your device.

## Which devices have been tested?

| Device | What we know today |
| --- | --- |
| Linux PC with a 64-bit Intel or AMD processor | The terminal runner ran offline, saved the fixture and reopened it successfully. Lubuntu is the development target. The oldest supported distribution and OS version are still open. |
| Windows PC with a 64-bit Intel or AMD processor | The runner built as a Windows executable. Launching it and reopening saves on Windows still need testing. |
| Mac with Apple silicon | The runner built as a native arm64 executable. Launching it and reopening saves on macOS still need testing. Intel Mac compatibility is unverified. |
| Phones, tablets and Chromebooks | A browser client is under consideration. The simulation's hash test passed under WebAssembly in Node. Browser versions, touch controls, save retention and offline startup on devices still need testing. |

## How much hardware will I need?

| Resource | Target for the finished game |
| --- | --- |
| Memory | **4 GB of available RAM** is the design target for a reduced, text-only town with AI turned off. **8 GB available** is the preferred target for a standard lightweight town. Both await gameplay benchmarks. |
| Processor and graphics | Ordinary play should work without a dedicated GPU. Minimum CPU speed, core count and graphics capabilities are still unmeasured. |
| Disk space | Desktop play needs writable space for local saves. Install size and typical save growth will be measured on packaged builds. |
| Internet and AI | The desktop game is designed for offline play after installation. AI is optional. A local model would have a separate memory and storage budget. |

To publish minimum and recommended specs, test a packaged build on each proposed OS and device. Record the OS version, hardware, available RAM, install size, save size and response time during a representative play session. Close the game, disconnect from the network, reopen the town and verify that progress survived. Browser tests also need to cover a browser restart after the first visit.

The [master design, section 4](design/Slop_Town_Master_Design_Revised_v7.docx) sets the hardware targets. The [Phase 0 plan](plans/phase-0.md) records the build and runtime evidence. Contributors building from source can find the Go toolchain requirements in the [README](../README.md#run-the-phase-0-fixture).
