<div class="grid grid-cols-2 gap-x-3 gap-y-5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6" aria-hidden="true">
	{#each { length: 6 } as _}
		<div>
			<div class="skeleton aspect-[4/3] rounded-xl"></div>
			<div class="space-y-1.5 pt-2">
				<div class="skeleton h-3.5 w-4/5"></div>
				<div class="skeleton h-3 w-1/2"></div>
			</div>
		</div>
	{/each}
</div>
