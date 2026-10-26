/**
 * Monotonic request guard for Composer restore paths. A restore may await
 * catalog, Agent, Workspace, Tool, or MCP work. Only the newest request may
 * commit after an await boundary.
 */
export class RestoreRequestGuard {
	private generation = 0;

	begin(): number {
		this.generation += 1;
		return this.generation;
	}

	invalidate(): void {
		this.generation += 1;
	}

	isCurrent(generation: number): boolean {
		return generation === this.generation;
	}
}
