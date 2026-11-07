<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5" aria-hidden="true">
	{#each { length: 6 } as _}
		<div class="card overflow-hidden">
			<div class="skeleton aspect-[4/3] rounded-none"></div>
			<div class="space-y-2 p-3">
				<div class="skeleton h-4 w-4/5"></div>
				<div class="skeleton h-3 w-1/2"></div>
			</div>
		</div>
	{/each}
</div>
