import createClient from "openapi-fetch";

import { paths } from "./api";

const client = createClient<paths>({
  baseUrl: "http://localhost:8000/rpc",
  headers: {
    "content-type": "application/json",
  },
});

declare global {
  interface Window {
    inc(): void;
  }
}

async function refetch() {
  const { data: count } = await client.POST("/counter/get", {});
  const countEl = document.querySelector("#counter");
  if (!countEl) {
    throw new Error("count element not found");
  }
  countEl.innerHTML = `${count}`;
}

window.inc = async () => {
  await client.POST("/counter/inc", {
    body: 1,
  });
  refetch();
};
